// Package engine is the CI/CD execution core: pipelines, runs, steps,
// persistence, and the Laya decision hooks. Ported from the original Python
// agent with two deliberate changes: JSON persistence instead of pickle, and
// AI analysis baked into failure/completion paths.
package engine

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// StepStatus enumerates the lifecycle of a pipeline step.
type StepStatus string

const (
	StepPending StepStatus = "pending"
	StepRunning StepStatus = "running"
	StepSuccess StepStatus = "success"
	StepFailed  StepStatus = "failed"
	StepSkipped StepStatus = "skipped"
)

// PipelineStatus enumerates the lifecycle of a pipeline run.
type PipelineStatus string

const (
	StatusPending   PipelineStatus = "pending"
	StatusRunning   PipelineStatus = "running"
	StatusSuccess   PipelineStatus = "success"
	StatusFailed    PipelineStatus = "failed"
	StatusCancelled PipelineStatus = "cancelled"
)

// Step is one command in a pipeline.
type Step struct {
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	Command         string          `json:"command"`
	Timeout         int             `json:"timeout"`           // seconds
	RetryCount      int             `json:"retry_count"`       // extra attempts
	ContinueOnError bool            `json:"continue_on_error"` // keep going after failure
	DependsOn       []string        `json:"depends_on,omitempty"`
	Artifacts       []string        `json:"artifacts,omitempty"`
	Status          StepStatus      `json:"status"`
	StartTime       string          `json:"start_time,omitempty"`
	EndTime         string          `json:"end_time,omitempty"`
	Output          string          `json:"output,omitempty"`
	Error           string          `json:"error,omitempty"`
	AIAnalysis      json.RawMessage `json:"ai_analysis,omitempty"` // filled by the Laya layer on failure
}

// PipelineRun is one execution of a pipeline definition.
type PipelineRun struct {
	ID            string         `json:"id"`
	PipelineID    string         `json:"pipeline_id"`
	Name          string         `json:"name"`
	Version       string         `json:"version"`
	Description   string         `json:"description,omitempty"`
	Variables     map[string]any `json:"variables,omitempty"`
	Steps         []Step         `json:"steps"`
	Status        PipelineStatus `json:"status"`
	CreatedAt     string         `json:"created_at"`
	StartedAt     *string        `json:"started_at,omitempty"`
	FinishedAt    *string        `json:"finished_at,omitempty"`
	TotalDuration *float64       `json:"total_duration,omitempty"`
}

// PipelineDefinition is the stored, user-authored pipeline.
type PipelineDefinition struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Description string         `json:"description,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
	Steps       []Step         `json:"steps"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

// Notification is a run-completion event routed by the Laya layer.
type Notification struct {
	ID                string  `json:"id"`
	Timestamp         string  `json:"ts"`
	RunID             string  `json:"run_id"`
	PipelineName      string  `json:"pipeline_name"`
	RunStatus         string  `json:"run_status"`
	EventType         string  `json:"event_type"`
	Title             string  `json:"title"`
	Message           string  `json:"message"`
	Level             string  `json:"level"`
	UrgentProbability float64 `json:"urgent_probability"`
}

// DecisionAnalyzer is the seam the Laya layer plugs into. Implementations
// must be safe for concurrent use and must never panic the engine: the
// engine recovers, but keep analysis cheap and quiet.
type DecisionAnalyzer interface {
	// AnalyzeStepFailure classifies a failed step and triages its output.
	AnalyzeStepFailure(step Step, runName string) json.RawMessage
	// NotifyRunCompletion decides how a finished run should be surfaced.
	NotifyRunCompletion(run PipelineRun) (level string, urgentProbability float64)
}

// Engine holds all state: definitions, active runs, history, notifications.
type Engine struct {
	mu            sync.RWMutex
	dataFile      string
	artifactsDir  string
	logs          *LogBroadcaster
	pipelines     map[string]*PipelineDefinition
	runs          map[string]*PipelineRun
	runHistory    []map[string]any
	notifications []Notification
	analyzer      DecisionAnalyzer
	cancels       map[string]context.CancelFunc
}

// New creates an Engine persisting to dataFile (JSON).
func New(dataFile string, analyzer DecisionAnalyzer) (*Engine, error) {
	artifactsDir := filepath.Join(filepath.Dir(dataFile), "artifacts")
	e := &Engine{
		dataFile:     dataFile,
		artifactsDir: artifactsDir,
		logs:         NewLogBroadcaster(1000),
		pipelines:    map[string]*PipelineDefinition{},
		runs:         map[string]*PipelineRun{},
		cancels:      map[string]context.CancelFunc{},
		analyzer:     analyzer,
	}
	if err := e.load(); err != nil {
		return nil, err
	}
	return e, nil
}

// Logs returns the log broadcaster for live streaming.
func (e *Engine) Logs() *LogBroadcaster {
	return e.logs
}

// ArtifactsDir returns the directory where run artifacts are stored.
func (e *Engine) ArtifactsDir() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.artifactsDir
}

// SetArtifactsDir allows configuring a custom artifacts storage directory.
func (e *Engine) SetArtifactsDir(dir string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.artifactsDir = dir
}

// --- persistence -----------------------------------------------------------

type persistedState struct {
	Pipelines     []PipelineDefinition `json:"pipelines"`
	Runs          []PipelineRun        `json:"runs"`
	RunHistory    []map[string]any     `json:"run_history"`
	Notifications []Notification       `json:"notifications"`
}

func (e *Engine) load() error {
	raw, err := os.ReadFile(e.dataFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read state: %w", err)
	}
	var state persistedState
	if err := json.Unmarshal(raw, &state); err != nil {
		// Corrupt state: back it up and start fresh (same policy as the
		// original Python agent).
		_ = os.Rename(e.dataFile, e.dataFile+".backup")
		return nil
	}
	for _, p := range state.Pipelines {
		p := p
		e.pipelines[p.ID] = &p
	}
	for i := range state.Runs {
		r := state.Runs[i]
		if r.Status == StatusRunning || r.Status == StatusPending {
			r.Status = StatusFailed
			now := time.Now().UTC().Format(time.RFC3339)
			r.FinishedAt = &now
			if r.StartedAt != nil {
				if t, err := time.Parse(time.RFC3339, *r.StartedAt); err == nil {
					d := time.Since(t).Seconds()
					r.TotalDuration = &d
				}
			}
			for si := range r.Steps {
				if r.Steps[si].Status == StepRunning || r.Steps[si].Status == StepPending {
					r.Steps[si].Status = StepFailed
					r.Steps[si].EndTime = now
					if r.Steps[si].Error == "" {
						r.Steps[si].Error = "execution interrupted by server shutdown"
					}
				}
			}
		}
		e.runs[r.ID] = &r
	}
	e.runHistory = state.RunHistory
	e.notifications = state.Notifications
	return nil
}

func (e *Engine) saveLocked() {
	state := persistedState{
		RunHistory:    e.runHistory,
		Notifications: e.notifications,
	}
	for _, p := range e.pipelines {
		state.Pipelines = append(state.Pipelines, *p)
	}
	for _, r := range e.runs {
		state.Runs = append(state.Runs, *r)
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(e.dataFile), 0o755)
	tmp := e.dataFile + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, e.dataFile)
}

// --- pipeline CRUD ----------------------------------------------------------

// CreatePipeline stores a new definition from config.
func (e *Engine) CreatePipeline(config PipelineDefinition) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	id := newID()
	now := time.Now().UTC().Format(time.RFC3339)
	def := config
	def.ID = id
	def.CreatedAt = now
	def.UpdatedAt = now
	def.Steps = withDefaultStepStatus(config.Steps)
	e.pipelines[id] = &def
	e.saveLocked()
	return id
}

func withDefaultStepStatus(steps []Step) []Step {
	for i := range steps {
		if steps[i].Status == "" {
			steps[i].Status = StepPending
		}
	}
	return steps
}

// DeletePipeline removes a definition (refuses while runs are active).
func (e *Engine) DeletePipeline(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.pipelines[id]; !ok {
		return false
	}
	for _, r := range e.runs {
		if r.PipelineID == id && r.Status == StatusRunning {
			return false
		}
	}
	delete(e.pipelines, id)
	e.saveLocked()
	return true
}

// ListPipelines returns definitions with their latest run status.
func (e *Engine) ListPipelines() []map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := []map[string]any{}
	for _, p := range e.pipelines {
		status := "never_run"
		var lastRunAt any
		active := 0
		total := 0
		for _, r := range e.runs {
			if r.PipelineID == p.ID {
				active++
				if r.Status == StatusRunning {
					status = string(r.Status)
				}
			}
		}
		for _, h := range e.runHistory {
			if h["pipeline_id"] == p.ID {
				total++
			}
		}
		if active == 0 {
			for i := len(e.runHistory) - 1; i >= 0; i-- {
				h := e.runHistory[i]
				if h["pipeline_id"] == p.ID {
					status, _ = h["status"].(string)
					lastRunAt = h["created_at"]
					break
				}
			}
		} else {
			for _, r := range e.runs {
				if r.PipelineID == p.ID {
					if r.StartedAt != nil {
						lastRunAt = *r.StartedAt
					} else {
						lastRunAt = r.CreatedAt
					}
					break
				}
			}
		}
		out = append(out, map[string]any{
			"id":          p.ID,
			"name":        p.Name,
			"description": p.Description,
			"status":      status,
			"created_at":  p.CreatedAt,
			"last_run_at": lastRunAt,
			"active_runs": active,
			"total_runs":  total + active,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i]["created_at"].(string) > out[j]["created_at"].(string)
	})
	return out
}

// GetPipeline returns a definition with its latest run info.
func (e *Engine) GetPipeline(id string) (map[string]any, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	p, ok := e.pipelines[id]
	if !ok {
		return nil, false
	}
	status := "never_run"
	var lastRunAt, lastFinishedAt, lastDuration any
	var steps any = stepsForView(p.Steps)
	for _, r := range e.runs {
		if r.PipelineID == id {
			status = string(r.Status)
			lastRunAt = r.StartedAt
			steps = stepsForView(r.Steps)
		}
	}
	if status == "never_run" {
		for i := len(e.runHistory) - 1; i >= 0; i-- {
			h := e.runHistory[i]
			if h["pipeline_id"] == id {
				status, _ = h["status"].(string)
				lastRunAt = h["started_at"]
				lastFinishedAt = h["finished_at"]
				lastDuration = h["total_duration"]
				if rawSteps, ok := h["steps"].([]any); ok {
					steps = rawSteps
				}
				break
			}
		}
	}
	return map[string]any{
		"id":               p.ID,
		"name":             p.Name,
		"version":          p.Version,
		"description":      p.Description,
		"variables":        p.Variables,
		"status":           status,
		"created_at":       p.CreatedAt,
		"last_run_at":      lastRunAt,
		"last_finished_at": lastFinishedAt,
		"total_duration":   lastDuration,
		"steps":            steps,
	}, true
}

func stepsForView(steps []Step) []map[string]any {
	out := make([]map[string]any, 0, len(steps))
	for _, s := range steps {
		out = append(out, map[string]any{
			"name":              s.Name,
			"description":       s.Description,
			"command":           s.Command,
			"timeout":           s.Timeout,
			"retry_count":       s.RetryCount,
			"continue_on_error": s.ContinueOnError,
			"depends_on":        s.DependsOn,
			"artifacts":         s.Artifacts,
			"status":            s.Status,
			"start_time":        s.StartTime,
			"end_time":          s.EndTime,
			"output":            s.Output,
			"error":             s.Error,
			"ai_analysis":       s.AIAnalysis,
		})
	}
	return out
}

// --- run lifecycle ------------------------------------------------------------

// CreateRun instantiates a run for a pipeline definition.
func (e *Engine) CreateRun(pipelineID string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	p, ok := e.pipelines[pipelineID]
	if !ok {
		return "", fmt.Errorf("pipeline %s not found", pipelineID)
	}
	id := newID()
	run := &PipelineRun{
		ID:          id,
		PipelineID:  pipelineID,
		Name:        p.Name,
		Version:     p.Version,
		Description: p.Description,
		Variables:   p.Variables,
		Steps:       withDefaultStepStatus(append([]Step(nil), p.Steps...)),
		Status:      StatusPending,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	e.runs[id] = run
	e.saveLocked()
	return id, nil
}

// StartRun executes a run asynchronously in its own goroutine.
// The returned runID is immediately queryable; execution continues in the
// background (status: running).
func (e *Engine) StartRun(pipelineID string, background bool) (string, error) {
	runID, err := e.CreateRun(pipelineID)
	if err != nil {
		return "", err
	}
	if !background {
		e.ExecuteRunSync(runID)
		return runID, nil
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				e.mu.Lock()
				if run, ok := e.runs[runID]; ok {
					run.Status = StatusFailed
					e.mu.Unlock()
					e.finalizeRun(runID)
				} else {
					e.mu.Unlock()
				}
			}
		}()
		e.ExecuteRunSync(runID)
	}()
	return runID, nil
}

// ExecuteRunSync runs steps according to dependency DAG (if defined) or sequentially; returns success.
func (e *Engine) ExecuteRunSync(runID string) bool {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	e.mu.Lock()
	run, ok := e.runs[runID]
	if !ok {
		e.mu.Unlock()
		return false
	}
	run.Status = StatusRunning
	started := time.Now().UTC().Format(time.RFC3339)
	run.StartedAt = &started
	e.cancels[runID] = cancel
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.cancels, runID)
		e.mu.Unlock()
	}()
	defer e.finalizeRun(runID)

	hasDAG := false
	for _, s := range run.Steps {
		if len(s.DependsOn) > 0 {
			hasDAG = true
			break
		}
	}

	if hasDAG {
		return e.executeRunDAG(ctx, run)
	}

	for i := range run.Steps {
		e.mu.RLock()
		cancelled := run.Status == StatusCancelled || errors.Is(ctx.Err(), context.Canceled)
		continueOnError := run.Steps[i].ContinueOnError
		e.mu.RUnlock()
		if cancelled {
			e.mu.Lock()
			run.Status = StatusCancelled
			for j := i; j < len(run.Steps); j++ {
				if run.Steps[j].Status == StepPending {
					run.Steps[j].Status = StepSkipped
				}
			}
			e.saveLocked()
			e.mu.Unlock()
			return false
		}

		ok := e.executeStep(ctx, run, i)
		if !ok {
			e.mu.Lock()
			isCancelled := run.Status == StatusCancelled || errors.Is(ctx.Err(), context.Canceled)
			if isCancelled {
				run.Status = StatusCancelled
			} else if !continueOnError {
				run.Status = StatusFailed
			}
			if isCancelled || !continueOnError {
				for j := i + 1; j < len(run.Steps); j++ {
					if run.Steps[j].Status == StepPending {
						run.Steps[j].Status = StepSkipped
					}
				}
				e.mu.Unlock()
				return false
			}
			e.mu.Unlock()
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if run.Status != StatusCancelled && run.Status != StatusFailed {
		run.Status = StatusSuccess
	}
	return run.Status == StatusSuccess
}

func (e *Engine) executeRunDAG(ctx context.Context, run *PipelineRun) bool {
	numSteps := len(run.Steps)
	nameToIdx := make(map[string]int, numSteps)
	for i, s := range run.Steps {
		if s.Name == "" {
			nameToIdx[fmt.Sprintf("step-%d", i)] = i
		} else {
			nameToIdx[s.Name] = i
		}
	}

	// Validate unknown dependencies
	for i, s := range run.Steps {
		for _, dep := range s.DependsOn {
			if _, exists := nameToIdx[dep]; !exists {
				e.mu.Lock()
				run.Status = StatusFailed
				run.Steps[i].Status = StepFailed
				run.Steps[i].Error = fmt.Sprintf("unknown dependency %q", dep)
				for j := range run.Steps {
					if j != i && run.Steps[j].Status == StepPending {
						run.Steps[j].Status = StepSkipped
					}
				}
				e.saveLocked()
				e.mu.Unlock()
				return false
			}
		}
	}

	// Cycle detection using Kahn's algorithm
	adj := make(map[int][]int)
	inDegree := make(map[int]int, numSteps)
	for i, s := range run.Steps {
		inDegree[i] = len(s.DependsOn)
		for _, dep := range s.DependsOn {
			depIdx := nameToIdx[dep]
			adj[depIdx] = append(adj[depIdx], i)
		}
	}

	queue := make([]int, 0, numSteps)
	for i := range run.Steps {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}
	visitedCount := 0
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		visitedCount++
		for _, next := range adj[curr] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if visitedCount < numSteps {
		// Cycle detected
		e.mu.Lock()
		run.Status = StatusFailed
		for i := range run.Steps {
			if run.Steps[i].Status == StepPending {
				run.Steps[i].Status = StepFailed
				run.Steps[i].Error = "cycle detected in step dependencies"
			}
		}
		e.saveLocked()
		e.mu.Unlock()
		return false
	}

	// Dynamic parallel scheduler
	stepStatus := make([]StepStatus, numSteps)
	for i := range stepStatus {
		stepStatus[i] = StepPending
	}

	stepFinished := make(chan int, numSteps)
	activeRunning := 0
	finishedCount := 0
	hasFailedWithoutContinue := false

	for finishedCount < numSteps {
		if errors.Is(ctx.Err(), context.Canceled) {
			e.mu.Lock()
			run.Status = StatusCancelled
			for i := range run.Steps {
				if run.Steps[i].Status == StepPending {
					run.Steps[i].Status = StepSkipped
				}
			}
			e.saveLocked()
			e.mu.Unlock()
			for activeRunning > 0 {
				<-stepFinished
				activeRunning--
			}
			return false
		}

		// Identify ready steps
		readyIndices := []int{}
		for i := 0; i < numSteps; i++ {
			if stepStatus[i] == StepPending {
				allDepsDone := true
				depFailed := false

				for _, dep := range run.Steps[i].DependsOn {
					depIdx := nameToIdx[dep]
					st := stepStatus[depIdx]
					if st == StepFailed {
						e.mu.RLock()
						canContinue := run.Steps[depIdx].ContinueOnError
						e.mu.RUnlock()
						if !canContinue {
							depFailed = true
							break
						}
					} else if st == StepSkipped {
						depFailed = true
						break
					} else if st != StepSuccess {
						allDepsDone = false
						break
					}
				}

				if depFailed || (hasFailedWithoutContinue && !run.Steps[i].ContinueOnError) {
					stepStatus[i] = StepSkipped
					e.mu.Lock()
					run.Steps[i].Status = StepSkipped
					run.Steps[i].EndTime = time.Now().UTC().Format(time.RFC3339)
					if run.Steps[i].Error == "" {
						run.Steps[i].Error = "skipped because dependency failed"
					}
					e.saveLocked()
					e.mu.Unlock()
					finishedCount++
				} else if allDepsDone {
					readyIndices = append(readyIndices, i)
				}
			}
		}

		// Launch ready steps concurrently
		for _, idx := range readyIndices {
			stepStatus[idx] = StepRunning
			activeRunning++
			go func(stepIdx int) {
				e.executeStep(ctx, run, stepIdx)
				stepFinished <- stepIdx
			}(idx)
		}

		if activeRunning == 0 {
			if finishedCount < numSteps {
				e.mu.Lock()
				for i := range run.Steps {
					if stepStatus[i] == StepPending {
						stepStatus[i] = StepSkipped
						run.Steps[i].Status = StepSkipped
						finishedCount++
					}
				}
				e.saveLocked()
				e.mu.Unlock()
			}
			break
		}

		select {
		case finishedIdx := <-stepFinished:
			activeRunning--
			finishedCount++
			e.mu.RLock()
			st := run.Steps[finishedIdx].Status
			cont := run.Steps[finishedIdx].ContinueOnError
			e.mu.RUnlock()
			stepStatus[finishedIdx] = st
			if st == StepFailed && !cont {
				hasFailedWithoutContinue = true
			}
		case <-ctx.Done():
			e.mu.Lock()
			run.Status = StatusCancelled
			for i := range run.Steps {
				if run.Steps[i].Status == StepPending {
					run.Steps[i].Status = StepSkipped
				}
			}
			e.saveLocked()
			e.mu.Unlock()
			for activeRunning > 0 {
				<-stepFinished
				activeRunning--
			}
			return false
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if run.Status != StatusCancelled {
		if hasFailedWithoutContinue {
			run.Status = StatusFailed
		} else {
			run.Status = StatusSuccess
		}
	}
	return run.Status == StatusSuccess
}

func (e *Engine) executeStep(ctx context.Context, run *PipelineRun, idx int) bool {
	e.mu.Lock()
	step := &run.Steps[idx]
	step.Status = StepRunning
	step.StartTime = time.Now().UTC().Format(time.RFC3339)
	command := substituteVariables(step.Command, run.Variables)
	timeout := step.Timeout
	retries := step.RetryCount
	runName := run.Name
	runID := run.ID
	analyzer := e.analyzer
	stepDesc := step.Description
	stepTimeout := step.Timeout
	stepRetries := step.RetryCount
	stepContinue := step.ContinueOnError
	stepName := step.Name
	stepArtifacts := append([]string(nil), step.Artifacts...)
	e.saveLocked()
	e.mu.Unlock()

	if e.logs != nil {
		e.logs.Publish(LogEvent{
			RunID:     runID,
			StepIndex: idx,
			StepName:  stepName,
			Stream:    "system",
			Line:      fmt.Sprintf("==> Starting step [%s]: %s", stepName, command),
		})
	}

	onLine := func(stream, line string) {
		if e.logs != nil {
			e.logs.Publish(LogEvent{
				RunID:     runID,
				StepIndex: idx,
				StepName:  stepName,
				Stream:    stream,
				Line:      line,
			})
		}
	}

	// shell execution with retry, timeout
	var lastOutput, lastErr string
	for attempt := 0; attempt <= retries; attempt++ {
		if ctx.Err() != nil {
			lastErr = "command execution cancelled"
			break
		}

		cmdOutput, cmdDetail, cmdErr := runCommand(ctx, command, timeout, onLine)
		lastOutput = cmdOutput
		if cmdErr == nil {
			if len(stepArtifacts) > 0 {
				_, _ = CollectArtifacts(e.artifactsDir, runID, stepName, stepArtifacts)
			}
			if e.logs != nil {
				e.logs.Publish(LogEvent{
					RunID:     runID,
					StepIndex: idx,
					StepName:  stepName,
					Stream:    "system",
					Line:      fmt.Sprintf("==> Step [%s] completed successfully", stepName),
				})
			}
			e.mu.Lock()
			step.Output = cmdOutput
			step.Status = StepSuccess
			step.EndTime = time.Now().UTC().Format(time.RFC3339)
			e.saveLocked()
			e.mu.Unlock()
			return true
		}
		lastErr = cmdDetail
		if attempt < retries {
			select {
			case <-ctx.Done():
				lastErr = "command execution cancelled"
				attempt = retries // stop retrying
			case <-time.After(time.Duration(1<<uint(attempt)) * time.Second):
			}
		}
	}

	isCancelled := errors.Is(ctx.Err(), context.Canceled)
	var analysis json.RawMessage
	if !isCancelled && analyzer != nil {
		// Run AI analysis outside the mutex lock to avoid blocking other concurrent requests
		analysis = analyzer.AnalyzeStepFailure(Step{
			Name:            stepName,
			Description:     stepDesc,
			Command:         command,
			Timeout:         stepTimeout,
			RetryCount:      stepRetries,
			ContinueOnError: stepContinue,
			Status:          StepFailed,
			Output:          lastOutput,
			Error:           lastErr,
		}, runName)

		// Flaky retry: if step had 0 retries and Laya classified this failure as flaky/transient,
		// attempt one automated flaky retry.
		if retries == 0 && analysis != nil && ctx.Err() == nil {
			var parsed struct {
				Classification struct {
					Kind  string `json:"kind"`
					Retry bool   `json:"retry"`
				} `json:"classification"`
			}
			if json.Unmarshal(analysis, &parsed) == nil && (parsed.Classification.Retry || parsed.Classification.Kind == "flaky") {
				if e.logs != nil {
					e.logs.Publish(LogEvent{
						RunID:     runID,
						StepIndex: idx,
						StepName:  stepName,
						Stream:    "system",
						Line:      fmt.Sprintf("==> Step [%s] classified as flaky (%s); performing automated retry...", stepName, parsed.Classification.Kind),
					})
				}
				retryOutput, retryDetail, retryErr := runCommand(ctx, command, timeout, onLine)
				if retryErr == nil {
					if len(stepArtifacts) > 0 {
						_, _ = CollectArtifacts(e.artifactsDir, runID, stepName, stepArtifacts)
					}
					if e.logs != nil {
						e.logs.Publish(LogEvent{
							RunID:     runID,
							StepIndex: idx,
							StepName:  stepName,
							Stream:    "system",
							Line:      fmt.Sprintf("==> Step [%s] recovered on flaky retry!", stepName),
						})
					}
					e.mu.Lock()
					step.Output = retryOutput
					step.Status = StepSuccess
					step.EndTime = time.Now().UTC().Format(time.RFC3339)
					step.AIAnalysis = analysis
					e.saveLocked()
					e.mu.Unlock()
					return true
				}
				lastOutput = retryOutput
				lastErr = retryDetail
			}
		}
	}

	if e.logs != nil {
		e.logs.Publish(LogEvent{
			RunID:     runID,
			StepIndex: idx,
			StepName:  stepName,
			Stream:    "system",
			Line:      fmt.Sprintf("==> Step [%s] failed: %s", stepName, lastErr),
		})
	}

	e.mu.Lock()
	step.Output = lastOutput
	step.Status = StepFailed
	step.EndTime = time.Now().UTC().Format(time.RFC3339)
	step.Error = lastErr
	if analysis != nil {
		step.AIAnalysis = analysis
	}
	e.saveLocked()
	e.mu.Unlock()
	return false
}

var varRegex = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}|\$([a-zA-Z_][a-zA-Z0-9_]*)`)

func substituteVariables(command string, vars map[string]any) string {
	if len(vars) == 0 {
		return command
	}
	return varRegex.ReplaceAllStringFunc(command, func(match string) string {
		var name string
		if strings.HasPrefix(match, "${") && strings.HasSuffix(match, "}") {
			name = match[2 : len(match)-1]
		} else if strings.HasPrefix(match, "$") {
			name = match[1:]
		}
		if val, ok := vars[name]; ok {
			return fmt.Sprintf("%v", val)
		}
		return match
	})
}

// CancelRun marks a run cancelled; the executor observes it before each step.
func (e *Engine) CancelRun(runID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	run, ok := e.runs[runID]
	if !ok {
		return false
	}
	run.Status = StatusCancelled
	if cancel, ok := e.cancels[runID]; ok {
		cancel()
	}
	e.saveLocked()
	return true
}

// CancelPipeline cancels every active run of a pipeline.
func (e *Engine) CancelPipeline(pipelineID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	anyCancelled := false
	for id, run := range e.runs {
		if run.PipelineID == pipelineID {
			run.Status = StatusCancelled
			if cancel, ok := e.cancels[id]; ok {
				cancel()
			}
			anyCancelled = true
		}
	}
	if anyCancelled {
		e.saveLocked()
	}
	return anyCancelled
}

func (e *Engine) finalizeRun(runID string) {
	e.mu.Lock()
	run, ok := e.runs[runID]
	if !ok {
		e.mu.Unlock()
		return
	}
	finished := time.Now().UTC().Format(time.RFC3339)
	run.FinishedAt = &finished
	if run.StartedAt != nil {
		if t, err := time.Parse(time.RFC3339, *run.StartedAt); err == nil {
			d := time.Since(t).Seconds()
			run.TotalDuration = &d
		}
	}
	runCopy := *run
	runCopy.Steps = append([]Step(nil), run.Steps...)
	analyzer := e.analyzer
	e.mu.Unlock()

	// Notify via the Laya layer outside the mutex lock
	var notification Notification
	var hasNotification bool
	if analyzer != nil {
		level, urgent := analyzer.NotifyRunCompletion(runCopy)
		failed := []string{}
		for _, s := range runCopy.Steps {
			if s.Status == StepFailed {
				failed = append(failed, s.Name)
			}
		}
		msg := "all steps passed"
		if len(failed) > 0 {
			msg = "failed steps: " + strings.Join(failed, ", ")
		}
		notification = Notification{
			ID:                newID(),
			Timestamp:         finished,
			RunID:             runCopy.ID,
			PipelineName:      runCopy.Name,
			RunStatus:         string(runCopy.Status),
			EventType:         "run_" + string(runCopy.Status),
			Title:             fmt.Sprintf("Pipeline '%s' %s", runCopy.Name, runCopy.Status),
			Message:           msg,
			Level:             level,
			UrgentProbability: urgent,
		}
		hasNotification = true
	}

	e.mu.Lock()
	if hasNotification {
		e.notifications = append(e.notifications, notification)
	}
	// move to history
	hist := map[string]any{}
	raw, _ := json.Marshal(run)
	_ = json.Unmarshal(raw, &hist)
	e.runHistory = append(e.runHistory, hist)
	delete(e.runs, runID)
	e.saveLocked()
	e.mu.Unlock()
}

// --- queries ---------------------------------------------------------------

// ListRuns returns run summaries (active + history), newest first.
func (e *Engine) ListRuns(pipelineID string) []map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := []map[string]any{}
	for _, r := range e.runs {
		if pipelineID == "" || r.PipelineID == pipelineID {
			out = append(out, runSummary(r))
		}
	}
	for _, h := range e.runHistory {
		if pipelineID == "" || h["pipeline_id"] == pipelineID {
			m := map[string]any{}
			for k, v := range h {
				m[k] = v
			}
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, _ := out[i]["created_at"].(string)
		b, _ := out[j]["created_at"].(string)
		return a > b
	})
	return out
}

func runSummary(r *PipelineRun) map[string]any {
	return map[string]any{
		"id":             r.ID,
		"pipeline_id":    r.PipelineID,
		"name":           r.Name,
		"status":         r.Status,
		"created_at":     r.CreatedAt,
		"started_at":     r.StartedAt,
		"finished_at":    r.FinishedAt,
		"total_duration": r.TotalDuration,
	}
}

// GetRun returns a full run view (steps included), searching active then history.
func (e *Engine) GetRun(runID string) (map[string]any, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if r, ok := e.runs[runID]; ok {
		return runView(r), true
	}
	for _, h := range e.runHistory {
		if h["id"] == runID {
			return h, true
		}
	}
	return nil, false
}

// DeleteRun removes a run (refuses while running).
func (e *Engine) DeleteRun(runID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r, ok := e.runs[runID]; ok {
		if r.Status == StatusRunning {
			return false
		}
		delete(e.runs, runID)
		if e.logs != nil {
			e.logs.ClearHistory(runID)
		}
		e.saveLocked()
		return true
	}
	for i, h := range e.runHistory {
		if h["id"] == runID {
			e.runHistory = append(e.runHistory[:i], e.runHistory[i+1:]...)
			if e.logs != nil {
				e.logs.ClearHistory(runID)
			}
			e.saveLocked()
			return true
		}
	}
	return false
}

// ListNotifications returns the most recent notifications.
func (e *Engine) ListNotifications() []Notification {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Notification, len(e.notifications))
	copy(out, e.notifications)
	// newest first
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func runView(r *PipelineRun) map[string]any {
	return map[string]any{
		"id":             r.ID,
		"pipeline_id":    r.PipelineID,
		"name":           r.Name,
		"status":         r.Status,
		"created_at":     r.CreatedAt,
		"started_at":     r.StartedAt,
		"finished_at":    r.FinishedAt,
		"total_duration": r.TotalDuration,
		"steps":          stepsForView(r.Steps),
	}
}

func newID() string {
	var b [2]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%d-%04x", time.Now().UnixMilli(), b)
}
