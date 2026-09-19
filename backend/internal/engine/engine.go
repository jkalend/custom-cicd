// Package engine is the CI/CD execution core: pipelines, runs, steps,
// persistence, and the Jev decision hooks. Ported from the original Python
// agent with two deliberate changes: JSON persistence instead of pickle, and
// AI analysis baked into failure/completion paths.
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	Status          StepStatus      `json:"status"`
	StartTime       string          `json:"start_time,omitempty"`
	EndTime         string          `json:"end_time,omitempty"`
	Output          string          `json:"output,omitempty"`
	Error           string          `json:"error,omitempty"`
	AIAnalysis      json.RawMessage `json:"ai_analysis,omitempty"` // filled by the Jev layer on failure
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

// Notification is a run-completion event routed by the Jev layer.
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

// DecisionAnalyzer is the seam the Jev layer plugs into. Implementations
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
	pipelines     map[string]*PipelineDefinition
	runs          map[string]*PipelineRun
	runHistory    []map[string]any
	notifications []Notification
	analyzer      DecisionAnalyzer
	cancels       map[string]context.CancelFunc
}

// New creates an Engine persisting to dataFile (JSON).
func New(dataFile string, analyzer DecisionAnalyzer) (*Engine, error) {
	e := &Engine{
		dataFile:  dataFile,
		pipelines: map[string]*PipelineDefinition{},
		runs:      map[string]*PipelineRun{},
		cancels:   map[string]context.CancelFunc{},
		analyzer:  analyzer,
	}
	if err := e.load(); err != nil {
		return nil, err
	}
	return e, nil
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
			for _, h := range e.runHistory {
				if h["pipeline_id"] == p.ID {
					status, _ = h["status"].(string)
					lastRunAt = h["created_at"]
					break
				}
			}
		} else {
			lastRunAt = nil
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
	var steps any = []map[string]any{}
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
		"description":      p.Description,
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

// ExecuteRunSync runs all steps sequentially; returns success.
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

	defer func() { delete(e.cancels, runID) }()
	defer e.finalizeRun(runID)

	start := time.Now()
	for i := range run.Steps {
		e.mu.RLock()
		cancelled := run.Status == StatusCancelled
		e.mu.RUnlock()
		if cancelled {
			break
		}
		ok := e.executeStep(ctx, run, i)
		if !ok && !run.Steps[i].ContinueOnError {
			e.mu.Lock()
			run.Status = StatusFailed
			e.mu.Unlock()
			return false
		}
	}
	e.mu.RLock()
	cancelled := run.Status == StatusCancelled
	e.mu.RUnlock()
	if !cancelled {
		e.mu.Lock()
		run.Status = StatusSuccess
		e.mu.Unlock()
	}
	_ = start
	return run.Status == StatusSuccess
}

func (e *Engine) executeStep(ctx context.Context, run *PipelineRun, idx int) bool {
	e.mu.Lock()
	step := &run.Steps[idx]
	step.Status = StepRunning
	step.StartTime = time.Now().UTC().Format(time.RFC3339)
	command := substituteVariables(step.Command, run.Variables)
	e.mu.Unlock()

	// shell execution with retry, timeout
	var lastErr string
	for attempt := 0; attempt <= step.RetryCount; attempt++ {
		cmdErr := runCommand(ctx, command, step.Timeout, step)
		if cmdErr == nil {
			e.mu.Lock()
			step.Status = StepSuccess
			step.EndTime = time.Now().UTC().Format(time.RFC3339)
			e.mu.Unlock()
			return true
		}
		lastErr = cmdErr.Error()
		if attempt < step.RetryCount {
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
		}
	}

	e.mu.Lock()
	step.Status = StepFailed
	step.EndTime = time.Now().UTC().Format(time.RFC3339)
	step.Error = lastErr
	if e.analyzer != nil && step.Status == StepFailed {
		analysis := e.analyzer.AnalyzeStepFailure(*step, run.Name)
		if analysis != nil {
			step.AIAnalysis = analysis
		}
	}
	e.mu.Unlock()
	return false
}

func substituteVariables(command string, vars map[string]any) string {
	if len(vars) == 0 {
		return command
	}
	// simple ${VAR} and $VAR substitution
	for k, v := range vars {
		s := fmt.Sprintf("%v", v)
		command = strings.ReplaceAll(command, "${"+k+"}", s)
		command = strings.ReplaceAll(command, "$"+k, s)
	}
	return command
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
	// Notify via the Jev layer (level decided by AI, appended to the log).
	var notification Notification
	if e.analyzer != nil {
		level, urgent := e.analyzer.NotifyRunCompletion(*run)
		failed := []string{}
		for _, s := range run.Steps {
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
			RunID:             run.ID,
			PipelineName:      run.Name,
			RunStatus:         string(run.Status),
			EventType:         "run_" + string(run.Status),
			Title:             fmt.Sprintf("Pipeline '%s' %s", run.Name, run.Status),
			Message:           msg,
			Level:             level,
			UrgentProbability: urgent,
		}
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
		e.saveLocked()
		return true
	}
	for i, h := range e.runHistory {
		if h["id"] == runID {
			e.runHistory = append(e.runHistory[:i], e.runHistory[i+1:]...)
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
	return fmt.Sprintf("%d-%04x", time.Now().UnixMilli(), time.Now().UnixNano()&0xffff)
}
