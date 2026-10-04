package laya

import (
	"path/filepath"
	"testing"

	"laya-cicd-backend/internal/engine"
)

func TestHeuristicAnswers(t *testing.T) {
	state := map[string]any{
		"pipeline": "build-api",
		"step":     "test",
		"error":    "exit status 1",
	}
	questions := map[string]QuestionSpec{
		"kind": {
			Type:           "choice",
			Instructions:   "What kind of failure?",
			ChoiceCriteria: failureKinds,
		},
		"retri": {
			Type:         "boolean",
			Instructions: "Retry?",
		},
		"severity": {
			Type:         "score",
			Instructions: "Severity?",
			ScoreCriteria: []string{
				"low",
				"medium",
				"high",
				"critical",
			},
		},
	}

	answers := heuristicAnswers(state, questions)
	if len(answers) != 3 {
		t.Fatalf("expected 3 answers, got %d", len(answers))
	}

	kindAns, ok := answers["kind"]
	if !ok || kindAns.Choice == "" {
		t.Errorf("expected choice answer for kind, got %+v", kindAns)
	}

	retriAns, ok := answers["retri"]
	if !ok || retriAns.Probability == nil {
		t.Errorf("expected probability answer for retri, got %+v", retriAns)
	}

	sevAns, ok := answers["severity"]
	if !ok || sevAns.Score == nil {
		t.Errorf("expected score answer for severity, got %+v", sevAns)
	}
}

func TestTailStrRuneSafe(t *testing.T) {
	str := "Log output: 🚀 Warning: ⚠️ test"
	got := tailStr(str, 8)
	if got != " ⚠️ test" {
		t.Errorf("tailStr = %q, want %q", got, " ⚠️ test")
	}
}

func TestAnalyzerOperations(t *testing.T) {
	tmpDir := t.TempDir()
	client := &Client{
		GatewayURL: "http://127.0.0.1:9999/v1/evaluate", // unreachable, triggers heuristic
		Model:      "convaiinnovations/laya",
		RecordFile: filepath.Join(tmpDir, "decisions.jsonl"),
	}
	analyzer := NewAnalyzer(client)

	// 1. Triage Logs
	triage := analyzer.TriageLogs("ERROR: database connection reset\npanic: nil pointer", "test-step")
	if triage["action"] == "" {
		t.Errorf("expected triage action, got: %+v", triage)
	}

	// 2. Route Issue
	issue := analyzer.RouteIssue("Crash on Windows 11", "System hangs when launching", []string{"desktop"})
	if issue["component"] == "" {
		t.Errorf("expected component, got: %+v", issue)
	}
	labels, ok := issue["labels"].([]string)
	if !ok || len(labels) == 0 {
		t.Errorf("expected routed labels, got: %+v", issue)
	}

	// 3. Analyze Step Failure
	blob := analyzer.AnalyzeStepFailure(engine.Step{
		Name:    "build",
		Command: "go build ./...",
		Error:   "exit status 1",
		Output:  "cannot find package",
		Timeout: 30,
	}, "test-pipeline")
	if len(blob) == 0 {
		t.Errorf("expected non-empty analysis blob")
	}

	// 4. Notify Run Completion
	level, prob := analyzer.NotifyRunCompletion(engine.PipelineRun{
		ID:     "run-1",
		Name:   "test-pipeline",
		Status: engine.StatusFailed,
		Steps: []engine.Step{
			{Name: "build", Status: engine.StepFailed},
		},
	})
	if level == "" {
		t.Errorf("expected notification level, got empty string")
	}
	_ = prob

	// 5. Verify decisions recorded and listed
	decisions := client.ListDecisions(10)
	if len(decisions) == 0 {
		t.Errorf("expected decisions to be recorded in %s", client.RecordFile)
	}

	// 6. Status check
	status := client.Status()
	if status["gateway"] != client.GatewayURL {
		t.Errorf("expected gateway URL %s, got %v", client.GatewayURL, status["gateway"])
	}
}
