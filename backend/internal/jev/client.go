// Package jev wraps the Vercel AI Gateway evaluate endpoint for the
// typesafe-ai/jev decision model, with a deterministic offline fallback.
//
// Wire protocol (POST {gateway}/v1/evaluate):
//
//	{"model":"typesafe-ai/jev", "state": {...}, "questions": {
//	    "name": {"type":"boolean|choice|score", "instructions":"...",
//	             "criteria": {...}|[...] }}}
//
// Answers come back under .answers.<name> as:
//
//	boolean -> {"probability": 0..1}
//	choice  -> {"choice": "key", "probabilities": {...}}
//	score   -> {"score": 2.75, "probabilities": {...}}
package jev

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	Provider      string             `json:"provider"` // "jev" | "heuristic"
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

	mu sync.Mutex // guards record file appends
}

// DefaultClient builds a client from environment and .env bootstrap.
func DefaultClient() *Client {
	bootstrapDotEnv()
	gateway := envOr("JEV_GATEWAY_URL", "https://ai-gateway.vercel.sh/v1/evaluate")
	record := envOr("JEV_RECORD_FILE", filepath.Join("data", "jev_decisions.jsonl"))
	return &Client{
		GatewayURL: gateway,
		Model:      envOr("JEV_MODEL", "typesafe-ai/jev"),
		APIKey:     firstEnv("AI_GATEWAY_API_KEY", "VERCEL_AI_GATEWAY_API_KEY", "TYPESAFE_API_KEY"),
		HTTP:       &http.Client{Timeout: 10 * time.Second},
		RecordFile: record,
	}
}

// Evaluate asks `questions` against `state`. When no API key is configured
// or the gateway fails, deterministic heuristic answers are returned so the
// engine never blocks on AI availability.
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
	if c.APIKey == "" {
		return nil, "heuristic", fmt.Errorf("no api key configured")
	}
	payload, err := json.Marshal(request{Model: c.Model, State: state, Questions: questions})
	if err != nil {
		return nil, "jev", err
	}
	req, err := http.NewRequest(http.MethodPost, c.GatewayURL, bytes.NewReader(payload))
	if err != nil {
		return nil, "jev", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "jev", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "jev", fmt.Errorf("gateway status %d", resp.StatusCode)
	}
	var body responseBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, "jev", err
	}
	out := map[string]Answer{}
	for name := range questions {
		raw, ok := body.Answers[name]
		if !ok {
			return nil, "jev", fmt.Errorf("gateway missing answer %q", name)
		}
		var ans Answer
		if err := json.Unmarshal(raw, &ans); err != nil {
			return nil, "jev", fmt.Errorf("decode answer %s: %w", name, err)
		}
		ans.Type = questions[name].Type
		ans.Provider = "jev"
		out[name] = ans
	}
	return out, "jev", nil
}

// Configured reports whether a gateway key is present.
func (c *Client) Configured() bool { return c.APIKey != "" }

// Status describes the Jev layer configuration for the API.
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
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
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
	raw, err := os.ReadFile(c.RecordFile)
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
