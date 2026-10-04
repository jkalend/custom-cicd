// Package laya is the decision layer: it wraps the local Laya gateway (the
// open-weights Laya decision models behind the /v1/evaluate dialect) with a
// deterministic offline fallback.
//
// Wire protocol (POST {gateway}/v1/evaluate):
//
//	{"model":"convaiinnovations/laya", "state": {...}, "questions": {
//	    "name": {"type":"boolean|choice|score", "instructions":"...",
//	             "criteria": {...}|[...] }}}
//
// Answers come back under .answers.<name> as:
//
//	boolean -> {"probability": 0..1}
//	choice  -> {"choice": "key", "probabilities": {...}}
//	score   -> {"score": 2.75, "probabilities": {...}}
package laya

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Question is one typed judgment in a request.
type Question struct {
	Type         string          `json:"type"` // "boolean" | "choice" | "score"
	Instructions string          `json:"instructions"`
	Criteria     json.RawMessage `json:"criteria,omitempty"` // object for choice, array for score
}

// QuestionSpec builds a Question from plain Go values (map for choice,
// slice for score, nil for boolean).
type QuestionSpec struct {
	Type           string
	Instructions   string
	ChoiceCriteria map[string]string // choice: option -> description
	ScoreCriteria  []string          // score: low -> high legend
}

func (s QuestionSpec) build() (Question, error) {
	q := Question{Type: s.Type, Instructions: s.Instructions}
	switch s.Type {
	case "choice":
		if len(s.ChoiceCriteria) == 0 {
			return q, fmt.Errorf("choice question needs criteria")
		}
		raw, err := json.Marshal(s.ChoiceCriteria)
		if err != nil {
			return q, err
		}
		q.Criteria = raw
	case "score":
		if len(s.ScoreCriteria) < 2 {
			return q, fmt.Errorf("score question needs >= 2 criteria")
		}
		raw, err := json.Marshal(s.ScoreCriteria)
		if err != nil {
			return q, err
		}
		q.Criteria = raw
	case "boolean":
		// no criteria
	default:
		return q, fmt.Errorf("unknown question type %q", s.Type)
	}
	return q, nil
}

// Answer is a normalized answer for one question.
type Answer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Probability   *float64           `json:"probability,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Provider      string             `json:"provider"` // "laya" | "heuristic"
}

// request is the wire shape.
type request struct {
	Model     string              `json:"model"`
	State     map[string]any      `json:"state"`
	Questions map[string]Question `json:"questions"`
}

// responseBody is the interesting subset of the gateway response.
type responseBody struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   map[string]any             `json:"usage"`
}

// Client talks to the gateway; safe for concurrent use.
type Client struct {
	GatewayURL string
	Model      string
	APIKey     string
	HTTP       *http.Client
	RecordFile string

	mu        sync.Mutex // guards record file appends and the health probe cache
	lastProbe time.Time
	probeOK   bool
}

// DefaultClient builds a client from environment and .env bootstrap.
func DefaultClient() *Client {
	bootstrapDotEnv()
	gateway := envOr("LAYA_GATEWAY_URL", "http://127.0.0.1:8128/v1/evaluate")
	record := envOr("LAYA_RECORD_FILE", filepath.Join("data", "laya_decisions.jsonl"))
	return &Client{
		GatewayURL: gateway,
		Model:      envOr("LAYA_MODEL", "convaiinnovations/laya"),
		APIKey:     envOr("LAYA_GATEWAY_TOKEN", ""),
		HTTP:       &http.Client{Timeout: 10 * time.Second},
		RecordFile: record,
	}
}
// Evaluate asks `questions` against `state`. When the gateway is unreachable
// or fails, deterministic heuristic answers are returned so the engine never
// blocks on AI availability.
func (c *Client) Evaluate(state map[string]any, questions map[string]QuestionSpec, context map[string]any) (map[string]Answer, error) {
	built := make(map[string]Question, len(questions))
	for name, spec := range questions {
		q, err := spec.build()
		if err != nil {
			return nil, fmt.Errorf("question %s: %w", name, err)
		}
		built[name] = q
	}

	started := time.Now()
	answers, provider, err := c.callGateway(state, built)
	if err != nil {
		answers = heuristicAnswers(state, questions)
		provider = "heuristic"
	}
	c.record(state, questions, answers, provider, time.Since(started), context)
	return answers, nil
}

func (c *Client) callGateway(state map[string]any, questions map[string]Question) (map[string]Answer, string, error) {
	// The local gateway needs no key; Authorization is only sent when one is
	// configured (a token-gated deployment via LAYA_GATEWAY_TOKEN).
	payload, err := json.Marshal(request{Model: c.Model, State: state, Questions: questions})
	if err != nil {
		return nil, "laya", err
	}
	req, err := http.NewRequest(http.MethodPost, c.GatewayURL, bytes.NewReader(payload))
	if err != nil {
		return nil, "laya", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "laya", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "laya", fmt.Errorf("gateway status %d", resp.StatusCode)
	}
	var body responseBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, "laya", err
	}
	out := map[string]Answer{}
	for name := range questions {
		raw, ok := body.Answers[name]
		// Local forwards can fail mid-load: a partial answer set is a whole-request
		// fallback, never a mixed result.
		if !ok {
			return nil, "laya", fmt.Errorf("gateway missing answer %q", name)
		}
		var ans Answer
		if err := json.Unmarshal(raw, &ans); err != nil {
			return nil, "laya", fmt.Errorf("decode answer %s: %w", name, err)
		}
		ans.Type = questions[name].Type
		ans.Provider = "laya"
		out[name] = ans
	}
	return out, "laya", nil
}

// Configured reports whether live evaluation is possible. The local gateway is
// keyless, so this is a reachability probe (GET /health, 2s budget) rather than
// a key check. Result is cached for a minute to keep the status endpoint cheap.
func (c *Client) Configured() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	ttl := 10 * time.Second
	if c.probeOK {
		ttl = time.Minute
	}
	if time.Since(c.lastProbe) < ttl && !c.lastProbe.IsZero() {
		return c.probeOK
	}
	c.lastProbe = time.Now()
	c.probeOK = false

	health := strings.TrimSuffix(c.GatewayURL, "/v1/evaluate") + "/health"
	probe, err := http.NewRequest(http.MethodGet, health, nil)
	if err != nil {
		return false
	}
	if c.APIKey != "" {
		probe.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	probeClient := &http.Client{Timeout: 2 * time.Second}
	resp, err := probeClient.Do(probe)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	c.probeOK = resp.StatusCode == http.StatusOK
	return c.probeOK
}

// Status describes the Laya layer configuration for the API.
func (c *Client) Status() map[string]any {
	return map[string]any{
		"configured": c.Configured(),
		"gateway":    c.GatewayURL,
		"model":      c.Model,
	}
}

// --- heuristic fallback ------------------------------------------------------

// heuristicAnswers derives stable pseudo-answers from state + question text,
// so the system works fully offline (no key, gateway down).
func heuristicAnswers(state map[string]any, questions map[string]QuestionSpec) map[string]Answer {
	stateJSON, _ := json.Marshal(state)
	out := map[string]Answer{}
	for name, spec := range questions {
		seed := name + "|" + spec.Instructions + "|" + string(stateJSON)
		h := fnv1a(seed)
		ans := Answer{Type: spec.Type, Provider: "heuristic"}
		switch spec.Type {
		case "boolean":
			p := float64(h) / float64(^uint32(0))
			ans.Probability = &p
		case "choice":
			options := make([]string, 0, len(spec.ChoiceCriteria))
			for k := range spec.ChoiceCriteria {
				options = append(options, k)
			}
			if len(options) == 0 {
				options = []string{"unknown"}
			}
			sortStrings(options)
			ans.Choice = options[int(h)%len(options)]
		case "score":
			n := len(spec.ScoreCriteria)
			s := float64(int(h) % n)
			ans.Score = &s
		}
		out[name] = ans
	}
	return out
}

func fnv1a(s string) uint32 {
	var h uint32 = 0x811C9DC5
	for i := range len(s) {
		h ^= uint32(s[i])
		h *= 0x01000193
	}
	return h
}

func sortStrings(s []string) {
	slices.Sort(s)
}

// --- decision recording ------------------------------------------------------

type recordEntry struct {
	ID        string            `json:"id"`
	TS        float64           `json:"ts"`
	Provider  string            `json:"provider"`
	LatencyMS float64           `json:"latency_ms"`
	Context   map[string]any    `json:"context"`
	State     map[string]any    `json:"state"`
	Questions map[string]any    `json:"questions"`
	Answers   map[string]Answer `json:"answers"`
	Error     string            `json:"error,omitempty"`
}

func (c *Client) record(state map[string]any, questions map[string]QuestionSpec, answers map[string]Answer, provider string, latency time.Duration, context map[string]any) {
	qs := map[string]any{}
	for name, spec := range questions {
		crit := any(nil)
		if spec.Type == "choice" {
			crit = spec.ChoiceCriteria
		} else if spec.Type == "score" {
			crit = spec.ScoreCriteria
		}
		qs[name] = map[string]any{
			"type":         spec.Type,
			"instructions": spec.Instructions,
			"criteria":     crit,
		}
	}
	entry := recordEntry{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		TS:        float64(time.Now().UnixMilli()) / 1000,
		Provider:  provider,
		LatencyMS: float64(latency.Milliseconds()),
		Context:   context,
		State:     state,
		Questions: qs,
		Answers:   answers,
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = os.MkdirAll(filepath.Dir(c.RecordFile), 0o755)
	f, err := os.OpenFile(c.RecordFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(raw, '\n'))
}

// ListDecisions reads the most recent decisions (newest first).
func (c *Client) ListDecisions(limit int) []map[string]any {
	if limit <= 0 {
		limit = 50
	}
	c.mu.Lock()
	raw, err := os.ReadFile(c.RecordFile)
	c.mu.Unlock()
	if err != nil {
		return []map[string]any{}
	}
	lines := bytes.Split(raw, []byte("\n"))
	out := []map[string]any{}
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		line := bytes.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(line, &m); err == nil {
			out = append(out, m)
		}
	}
	return out
}

// SaveFeedback appends user feedback (rating, comment, decision_id) to laya_feedback.jsonl.
func (c *Client) SaveFeedback(feedback map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	feedbackFile := filepath.Join(filepath.Dir(c.RecordFile), "laya_feedback.jsonl")
	_ = os.MkdirAll(filepath.Dir(feedbackFile), 0o755)

	if _, ok := feedback["id"]; !ok {
		feedback["id"] = strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	if _, ok := feedback["ts"]; !ok {
		feedback["ts"] = float64(time.Now().UnixMilli()) / 1000
	}

	raw, err := json.Marshal(feedback)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(feedbackFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(raw, '\n'))
	return err
}

// ListFeedback reads the most recent user feedback entries (newest first).
func (c *Client) ListFeedback(limit int) []map[string]any {
	if limit <= 0 {
		limit = 50
	}
	c.mu.Lock()
	feedbackFile := filepath.Join(filepath.Dir(c.RecordFile), "laya_feedback.jsonl")
	raw, err := os.ReadFile(feedbackFile)
	c.mu.Unlock()
	if err != nil {
		return []map[string]any{}
	}
	lines := bytes.Split(raw, []byte("\n"))
	out := []map[string]any{}
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		line := bytes.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(line, &m); err == nil {
			out = append(out, m)
		}
	}
	return out
}

// --- env helpers ---------------------------------------------------------------

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func firstEnv(names ...string) string {
	for _, n := range names {
		if v := os.Getenv(n); v != "" {
			return v
		}
	}
	return ""
}

// bootstrapDotEnv reads KEY=VALUE pairs from backend/.env without requiring
// an external dependency. Existing environment wins.
func bootstrapDotEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		k, v, ok := bytes.Cut(line, []byte("="))
		if !ok {
			continue
		}
		key := string(bytes.TrimSpace(k))
		val := string(bytes.TrimSpace(v))
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}
