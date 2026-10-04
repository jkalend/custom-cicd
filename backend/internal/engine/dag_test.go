package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEngineDAGExecution_Success(t *testing.T) {
	dataFile := filepath.Join(t.TempDir(), "test_state.json")
	eng, err := New(dataFile, nil)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	pID := eng.CreatePipeline(PipelineDefinition{
		Name: "dag-pipeline",
		Steps: []Step{
			{Name: "setup", Command: "echo setup"},
			{Name: "build-backend", Command: "echo backend", DependsOn: []string{"setup"}},
			{Name: "build-frontend", Command: "echo frontend", DependsOn: []string{"setup"}},
			{Name: "deploy", Command: "echo deploy", DependsOn: []string{"build-backend", "build-frontend"}},
		},
	})

	runID, err := eng.StartRun(pID, false)
	if err != nil {
		t.Fatalf("failed to start run: %v", err)
	}

	run, ok := eng.GetRun(runID)
	if !ok {
		t.Fatalf("run not found")
	}

	if run["status"] != string(StatusSuccess) {
		t.Fatalf("expected run status %s, got %v", StatusSuccess, run["status"])
	}

	steps := extractSteps(run["steps"])
	if len(steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(steps))
	}
	for _, s := range steps {
		if s["status"] != string(StepSuccess) {
			t.Errorf("expected step %v to succeed, got %v", s["name"], s["status"])
		}
	}
}

func TestEngineDAGExecution_CycleDetection(t *testing.T) {
	dataFile := filepath.Join(t.TempDir(), "test_state.json")
	eng, err := New(dataFile, nil)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	pID := eng.CreatePipeline(PipelineDefinition{
		Name: "cyclic-pipeline",
		Steps: []Step{
			{Name: "step-a", Command: "echo A", DependsOn: []string{"step-b"}},
			{Name: "step-b", Command: "echo B", DependsOn: []string{"step-a"}},
		},
	})

	runID, err := eng.StartRun(pID, false)
	if err != nil {
		t.Fatalf("failed to start run: %v", err)
	}

	run, ok := eng.GetRun(runID)
	if !ok {
		t.Fatalf("run not found")
	}

	if run["status"] != string(StatusFailed) {
		t.Fatalf("expected cyclic pipeline to fail, got %v", run["status"])
	}
}

func TestEngineDAGExecution_UnknownDependency(t *testing.T) {
	dataFile := filepath.Join(t.TempDir(), "test_state.json")
	eng, err := New(dataFile, nil)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	pID := eng.CreatePipeline(PipelineDefinition{
		Name: "bad-dep-pipeline",
		Steps: []Step{
			{Name: "step-a", Command: "echo A", DependsOn: []string{"missing-step"}},
		},
	})

	runID, err := eng.StartRun(pID, false)
	if err != nil {
		t.Fatalf("failed to start run: %v", err)
	}

	run, ok := eng.GetRun(runID)
	if !ok {
		t.Fatalf("run not found")
	}

	if run["status"] != string(StatusFailed) {
		t.Fatalf("expected unknown dependency pipeline to fail, got %v", run["status"])
	}
}

func TestEngineDAGExecution_CascadeSkip(t *testing.T) {
	dataFile := filepath.Join(t.TempDir(), "test_state.json")
	eng, err := New(dataFile, nil)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	pID := eng.CreatePipeline(PipelineDefinition{
		Name: "failing-dag",
		Steps: []Step{
			{Name: "failing-step", Command: "exit 1"},
			{Name: "dependent-step", Command: "echo won't run", DependsOn: []string{"failing-step"}},
		},
	})

	runID, err := eng.StartRun(pID, false)
	if err != nil {
		t.Fatalf("failed to start run: %v", err)
	}

	run, ok := eng.GetRun(runID)
	if !ok {
		t.Fatalf("run not found")
	}

	if run["status"] != string(StatusFailed) {
		t.Fatalf("expected run status %s, got %v", StatusFailed, run["status"])
	}

	steps := extractSteps(run["steps"])
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0]["status"] != string(StepFailed) {
		t.Errorf("step 0 status: %v, want %s", steps[0]["status"], StepFailed)
	}
	if steps[1]["status"] != string(StepSkipped) {
		t.Errorf("step 1 status: %v, want %s", steps[1]["status"], StepSkipped)
	}
}

func TestLogBroadcasterAndArtifacts(t *testing.T) {
	// Test LogBroadcaster
	b := NewLogBroadcaster(10)
	runID := "run-123"

	b.Publish(LogEvent{RunID: runID, StepName: "init", Stream: "stdout", Line: "hello 1"})
	b.Publish(LogEvent{RunID: runID, StepName: "init", Stream: "stdout", Line: "hello 2"})

	history, ch, unsub := b.Subscribe(runID)
	defer unsub()

	if len(history) != 2 {
		t.Fatalf("expected 2 history events, got %d", len(history))
	}

	b.Publish(LogEvent{RunID: runID, StepName: "init", Stream: "stdout", Line: "hello 3"})

	select {
	case ev := <-ch:
		if ev.Line != "hello 3" {
			t.Errorf("expected 'hello 3', got %q", ev.Line)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for live log event")
	}

	b.ClearHistory(runID)
	h2, _, unsub2 := b.Subscribe(runID)
	unsub2()
	if len(h2) != 0 {
		t.Fatalf("expected 0 history events after clear, got %d", len(h2))
	}

	// Test Artifacts
	tmpDir := t.TempDir()
	artStorage := filepath.Join(tmpDir, "storage")
	srcFile := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(srcFile, []byte(`{"coverage": 95}`), 0o644); err != nil {
		t.Fatalf("failed to create src file: %v", err)
	}

	collected, err := CollectArtifacts(artStorage, "run-art-1", "test-step", []string{srcFile})
	if err != nil {
		t.Fatalf("CollectArtifacts failed: %v", err)
	}
	if len(collected) != 1 || collected[0].Name != "report.json" {
		t.Fatalf("unexpected collected artifacts: %+v", collected)
	}

	listed, err := ListRunArtifacts(artStorage, "run-art-1")
	if err != nil {
		t.Fatalf("ListRunArtifacts failed: %v", err)
	}
	if len(listed) != 1 || listed[0].Name != "report.json" {
		t.Fatalf("unexpected listed artifacts: %+v", listed)
	}
}
