package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSubstituteVariables(t *testing.T) {
	vars := map[string]any{
		"APP":     "production-api",
		"APP_ENV": "staging",
		"PORT":    8080,
	}

	cmd := "deploy --name $APP_ENV --service $APP --port ${PORT}"
	got := substituteVariables(cmd, vars)
	want := "deploy --name staging --service production-api --port 8080"
	if got != want {
		t.Errorf("substituteVariables = %q, want %q", got, want)
	}
}

func TestRunCommandOutput(t *testing.T) {
	ctx := context.Background()
	output, detail, err := runCommand(ctx, "echo hello", 10, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail != "" {
		t.Errorf("unexpected detail on success: %q", detail)
	}
	if output == "" {
		t.Errorf("expected non-empty output")
	}
}

func TestTailUTF8Safe(t *testing.T) {
	// Multi-byte string with emojis and non-ASCII runes
	str := "Pipeline 🚀 failed with error: čučoriedka 🧠"
	t1 := tail(str, 5)
	if t1 != "dka 🧠" {
		t.Errorf("tail = %q, want %q", t1, "dka 🧠")
	}
}

func TestEngineLifecycleAndLatestRun(t *testing.T) {
	tmpDir := t.TempDir()
	dataFile := filepath.Join(tmpDir, "test_data.json")

	eng, err := New(dataFile, nil)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	pID := eng.CreatePipeline(PipelineDefinition{
		Name: "Test Pipeline",
		Steps: []Step{
			{Name: "step1", Command: "echo step 1", Timeout: 10},
		},
	})

	// Run 1
	runID1, err := eng.StartRun(pID, false)
	if err != nil {
		t.Fatalf("start run 1 failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	// Run 2
	runID2, err := eng.StartRun(pID, false)
	if err != nil {
		t.Fatalf("start run 2 failed: %v", err)
	}

	pipelines := eng.ListPipelines()
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}

	// Verify ListPipelines reflects the latest run status (not the oldest)
	if pipelines[0]["status"] != "success" {
		t.Errorf("expected status 'success', got %v", pipelines[0]["status"])
	}

	// Verify runs are retrieved
	r1, ok1 := eng.GetRun(runID1)
	if !ok1 || r1 == nil {
		t.Errorf("expected run 1 in history")
	}
	r2, ok2 := eng.GetRun(runID2)
	if !ok2 || r2 == nil {
		t.Errorf("expected run 2 in history")
	}

	// Test recovery on restart: write simulated running run to file
	eng.mu.Lock()
	eng.runs["zombie-1"] = &PipelineRun{
		ID:         "zombie-1",
		PipelineID: pID,
		Name:       "Zombie Run",
		Status:     StatusRunning,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Steps: []Step{
			{Name: "hanging", Status: StepRunning},
		},
	}
	eng.saveLocked()
	eng.mu.Unlock()

	// Reload engine from file
	eng2, err := New(dataFile, nil)
	if err != nil {
		t.Fatalf("failed to reload engine: %v", err)
	}

	zombie, ok := eng2.GetRun("zombie-1")
	if !ok {
		t.Fatalf("zombie run not found after reload")
	}
	if zombie["status"] != StatusFailed {
		t.Errorf("expected zombie run status 'failed', got %v", zombie["status"])
	}

	_ = os.Remove(dataFile)
}

func TestCancelRun(t *testing.T) {
	tmpDir := t.TempDir()
	dataFile := filepath.Join(tmpDir, "test_cancel.json")

	eng, err := New(dataFile, nil)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	pID := eng.CreatePipeline(PipelineDefinition{
		Name: "Cancel Pipeline",
		Steps: []Step{
			{Name: "slow_step", Command: "ping 127.0.0.1 -n 5 || sleep 5", Timeout: 10},
			{Name: "next_step", Command: "echo should be skipped", Timeout: 10},
		},
	})

	runID, err := eng.StartRun(pID, true)
	if err != nil {
		t.Fatalf("failed to start run: %v", err)
	}

	// Give the step a moment to transition to running
	time.Sleep(100 * time.Millisecond)

	if !eng.CancelRun(runID) {
		t.Fatalf("CancelRun returned false")
	}

	// Wait for background execution to finalize
	deadline := time.Now().Add(3 * time.Second)
	var finalRun map[string]any
	for time.Now().Before(deadline) {
		r, ok := eng.GetRun(runID)
		if ok {
			status, _ := r["status"].(string)
			if status == string(StatusCancelled) || status == string(StatusFailed) {
				finalRun = r
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	if finalRun == nil {
		t.Fatalf("run did not finalize after cancel")
	}

	if finalRun["status"] != string(StatusCancelled) {
		t.Errorf("expected run status %q, got %q", StatusCancelled, finalRun["status"])
	}

	steps := extractSteps(finalRun["steps"])
	if len(steps) < 2 {
		t.Fatalf("expected 2 steps in run, got: %+v", finalRun["steps"])
	}
	if steps[1]["status"] != StepSkipped && steps[1]["status"] != string(StepSkipped) {
		t.Errorf("expected step 2 status %q, got %q", StepSkipped, steps[1]["status"])
	}
}

func TestContinueOnError(t *testing.T) {
	tmpDir := t.TempDir()
	dataFile := filepath.Join(tmpDir, "test_continue.json")

	eng, err := New(dataFile, nil)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	pID := eng.CreatePipeline(PipelineDefinition{
		Name: "Continue On Error Pipeline",
		Steps: []Step{
			{Name: "failing_step", Command: "exit 1 || (call)", Timeout: 10, ContinueOnError: true},
			{Name: "succeeding_step", Command: "echo ok", Timeout: 10},
		},
	})

	runID, err := eng.StartRun(pID, false)
	if err != nil {
		t.Fatalf("failed to start run: %v", err)
	}

	run, ok := eng.GetRun(runID)
	if !ok {
		t.Fatalf("expected run to exist")
	}

	if run["status"] != string(StatusSuccess) {
		t.Errorf("expected run status %q with continueOnError, got %q", StatusSuccess, run["status"])
	}

	steps := extractSteps(run["steps"])
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps in run, got: %+v", run["steps"])
	}
	if steps[0]["status"] != StepFailed && steps[0]["status"] != string(StepFailed) {
		t.Errorf("expected step 1 to be failed, got %q", steps[0]["status"])
	}
	if steps[1]["status"] != StepSuccess && steps[1]["status"] != string(StepSuccess) {
		t.Errorf("expected step 2 to be success, got %q", steps[1]["status"])
	}
}

func extractSteps(raw any) []map[string]any {
	switch v := raw.(type) {
	case []map[string]any:
		return v
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

func TestSubstituteVariablesEdgeCases(t *testing.T) {
	vars := map[string]any{
		"FOO": "bar",
	}

	// Unknown variable should remain untouched
	cmd1 := "echo $FOO and $UNKNOWN"
	got1 := substituteVariables(cmd1, vars)
	want1 := "echo bar and $UNKNOWN"
	if got1 != want1 {
		t.Errorf("substituteVariables = %q, want %q", got1, want1)
	}

	// Escaped or non-identifier shouldn't be altered
	cmd2 := "echo $$ and $?"
	got2 := substituteVariables(cmd2, vars)
	want2 := "echo $$ and $?"
	if got2 != want2 {
		t.Errorf("substituteVariables = %q, want %q", got2, want2)
	}
}
