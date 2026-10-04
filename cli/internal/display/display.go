package display

import (
	"fmt"
	"strings"
	"time"

	"custom-cicd-cli/internal/client"
)

// StatusEmojis maps status strings to emoji representations
var StatusEmojis = map[string]string{
	"pending":   "⏳",
	"running":   "🚀", 
	"success":   "✅",
	"failed":    "❌",
	"cancelled": "🛑",
	"skipped":   "⏭️",
	"never_run": "💤",
}

// PrintPipelines displays a list of pipelines in a formatted way
func PrintPipelines(pipelines []client.Pipeline) {
	if len(pipelines) == 0 {
		fmt.Println("📋 No pipelines found")
		return
	}

	fmt.Printf("\n📋 Found %d pipeline(s):\n", len(pipelines))
	fmt.Println(strings.Repeat("-", 80))
	
	for _, pipeline := range pipelines {
		emoji := StatusEmojis[pipeline.Status]
		if emoji == "" {
			emoji = "❓"
		}

		fmt.Printf("%s %s\n", emoji, pipeline.Name)
		fmt.Printf("\tID: %s\n", pipeline.ID)
		fmt.Printf("\tStatus: %s\n", pipeline.Status)
		fmt.Printf("\tCreated: %s\n", pipeline.CreatedAt)
		if pipeline.StartedAt != nil {
			fmt.Printf("\tStarted: %s\n", *pipeline.StartedAt)
		}
		if pipeline.FinishedAt != nil {
			fmt.Printf("\tFinished: %s\n", *pipeline.FinishedAt)
		}
		fmt.Println()
	}
}

// PrintPipelineDetails displays detailed information about a pipeline
func PrintPipelineDetails(pipeline *client.Pipeline) {
	emoji := StatusEmojis[pipeline.Status]
	if emoji == "" {
		emoji = "❓"
	}

	fmt.Printf("\n%s Pipeline: %s\n", emoji, pipeline.Name)
	fmt.Printf("📋 ID: %s\n", pipeline.ID)
	fmt.Printf("📊 Status: %s\n", pipeline.Status)
	if pipeline.StartedAt != nil {
		fmt.Printf("🕐 Started: %s\n", *pipeline.StartedAt)
	}
	if pipeline.FinishedAt != nil {
		fmt.Printf("🕐 Finished: %s\n", *pipeline.FinishedAt)
	}
	if pipeline.Duration != nil {
		fmt.Printf("⏱️  Duration: %.2f seconds\n", *pipeline.Duration)
	}

	if len(pipeline.Steps) > 0 {
		fmt.Println("\n📝 Steps:")
		for i, step := range pipeline.Steps {
			stepEmoji := StatusEmojis[step.Status]
			if stepEmoji == "" {
				stepEmoji = "❓"
			}

			fmt.Printf("  %d. %s %s [%s]\n", i+1, stepEmoji, step.Name, step.Status)
			if step.Output != "" && strings.TrimSpace(step.Output) != "" {
				fmt.Printf("     📤 Output: %s\n", strings.TrimSpace(step.Output))
			}
			if step.Error != "" && strings.TrimSpace(step.Error) != "" {
				fmt.Printf("     🚨 Error: %s\n", strings.TrimSpace(step.Error))
			}
		}
	}
}

// PrintRuns displays a list of runs in a formatted way
func PrintRuns(runs []client.Run) {
	if len(runs) == 0 {
		fmt.Println("🏃 No runs found")
		return
	}

	fmt.Printf("\n🏃 Found %d run(s):\n", len(runs))
	fmt.Println(strings.Repeat("-", 80))

	for _, run := range runs {
		emoji := StatusEmojis[run.Status]
		if emoji == "" {
			emoji = "❓"
		}

		fmt.Printf("%s %s\n", emoji, run.Name)
		fmt.Printf("   Run ID: %s\n", run.ID)
		fmt.Printf("   Pipeline ID: %s\n", run.PipelineID)
		fmt.Printf("   Status: %s\n", run.Status)
		fmt.Printf("   Created: %s\n", run.CreatedAt)
		if run.StartedAt != nil {
			fmt.Printf("   Started: %s\n", *run.StartedAt)
		}
		if run.FinishedAt != nil {
			fmt.Printf("   Finished: %s\n", *run.FinishedAt)
		}
		if run.Duration != nil {
			fmt.Printf("   Duration: %.2fs\n", *run.Duration)
		}
		fmt.Println()
	}
}

// PrintRunDetails displays detailed information about a run
func PrintRunDetails(run *client.Run) {
	emoji := StatusEmojis[run.Status]
	if emoji == "" {
		emoji = "❓"
	}

	fmt.Printf("\n%s Run: %s\n", emoji, run.Name)
	fmt.Printf("🏃 Run ID: %s\n", run.ID)
	fmt.Printf("📋 Pipeline ID: %s\n", run.PipelineID)
	fmt.Printf("📊 Status: %s\n", run.Status)
	fmt.Printf("🕐 Created: %s\n", run.CreatedAt)
	if run.StartedAt != nil {
		fmt.Printf("🕐 Started: %s\n", *run.StartedAt)
	}
	if run.FinishedAt != nil {
		fmt.Printf("🕐 Finished: %s\n", *run.FinishedAt)
	}
	if run.Duration != nil {
		fmt.Printf("⏱️  Duration: %.2f seconds\n", *run.Duration)
	}

	if len(run.Steps) > 0 {
		fmt.Println("\n📝 Steps:")
		for i, step := range run.Steps {
			stepEmoji := StatusEmojis[step.Status]
			if stepEmoji == "" {
				stepEmoji = "❓"
			}

			fmt.Printf("  %d. %s %s [%s]\n", i+1, stepEmoji, step.Name, step.Status)
			if step.Output != "" && strings.TrimSpace(step.Output) != "" {
				fmt.Printf("     📤 Output: %s\n", strings.TrimSpace(step.Output))
			}
			if step.Error != "" && strings.TrimSpace(step.Error) != "" {
				fmt.Printf("     🚨 Error: %s\n", strings.TrimSpace(step.Error))
			}
		}
	}
}

// PrintSuccess prints a success message with emoji
func PrintSuccess(message string) {
	fmt.Printf("✅ %s\n", message)
}

// PrintError prints an error message with emoji
func PrintError(message string) {
	fmt.Printf("❌ %s\n", message)
}

// PrintInfo prints an info message with emoji
func PrintInfo(message string) {
	fmt.Printf("ℹ️  %s\n", message)
}

// PrintWarning prints a warning message with emoji
func PrintWarning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}

// FormatDuration formats a duration in seconds to a human-readable string
func FormatDuration(seconds float64) string {
	duration := time.Duration(seconds * float64(time.Second))
	if duration < time.Minute {
		return fmt.Sprintf("%.1fs", duration.Seconds())
	}
	if duration < time.Hour {
		return fmt.Sprintf("%.1fm", duration.Minutes())
	}
	return fmt.Sprintf("%.1fh", duration.Hours())
}

// PrintArtifacts displays a list of captured artifacts for a run
func PrintArtifacts(artifacts []client.Artifact) {
	if len(artifacts) == 0 {
		fmt.Println("📦 No artifacts found for this run")
		return
	}

	fmt.Printf("\n📦 Found %d artifact(s):\n", len(artifacts))
	fmt.Println(strings.Repeat("-", 60))

	for _, art := range artifacts {
		sizeKB := float64(art.Size) / 1024.0
		fmt.Printf("📄 %s\n", art.Name)
		fmt.Printf("\tStep: %s\n", art.StepName)
		fmt.Printf("\tSize: %.1f KB (%d bytes)\n", sizeKB, art.Size)
		fmt.Printf("\tPath: %s\n", art.Path)
		fmt.Println()
	}
}

