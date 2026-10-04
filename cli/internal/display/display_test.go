package display

import (
	"testing"

	"custom-cicd-cli/internal/client"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{5.2, "5.2s"},
		{65.0, "1.1m"},
		{3665.0, "1.0h"},
	}

	for _, tt := range tests {
		got := FormatDuration(tt.seconds)
		if got != tt.want {
			t.Errorf("FormatDuration(%f) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestStatusEmojis(t *testing.T) {
	expected := []string{"pending", "running", "success", "failed", "cancelled", "skipped", "never_run"}
	for _, status := range expected {
		if emoji, ok := StatusEmojis[status]; !ok || emoji == "" {
			t.Errorf("expected emoji for status %q", status)
		}
	}
}

func TestPrintHelpersNoPanic(t *testing.T) {
	// Ensure print functions handle empty or populated structs without panic
	PrintPipelines(nil)
	dur := 12.34
	started := "2026-10-03T18:00:00Z"
	finished := "2026-10-03T18:00:12Z"
	PrintPipelines([]client.Pipeline{
		{
			ID:         "p-1",
			Name:       "Test",
			Status:     "success",
			CreatedAt:  "2026-10-03T18:00:00Z",
			StartedAt:  &started,
			FinishedAt: &finished,
			Duration:   &dur,
			Steps: []client.Step{
				{Name: "step1", Command: "echo hi", Status: "success", Output: "hi"},
			},
		},
	})

	PrintPipelineDetails(&client.Pipeline{
		ID:       "p-1",
		Name:     "Test",
		Status:   "failed",
		Duration: &dur,
		Steps: []client.Step{
			{Name: "step1", Command: "echo fail", Status: "failed", Error: "err"},
		},
	})

	PrintRuns(nil)
	PrintRuns([]client.Run{
		{
			ID:         "r-1",
			PipelineID: "p-1",
			Name:       "Run 1",
			Status:     "running",
			CreatedAt:  "2026-10-03T18:00:00Z",
		},
	})

	PrintRunDetails(&client.Run{
		ID:         "r-1",
		PipelineID: "p-1",
		Name:       "Run 1",
		Status:     "success",
		Steps: []client.Step{
			{Name: "step1", Status: "success", Output: "done"},
		},
	})

	PrintSuccess("success")
	PrintError("error")
	PrintInfo("info")
	PrintWarning("warning")
}
