package cmd

import (
	"fmt"
	"time"

	"custom-cicd-cli/internal/display"

	"github.com/spf13/cobra"
)

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor <pipeline-id-or-run-id>",
	Short: "Monitor a pipeline or run in real-time",
	Long: `Monitor a pipeline or run's progress in real-time.
Updates every 2 seconds by default. Press Ctrl+C to stop monitoring.

The command will automatically detect if the provided ID is a pipeline or run ID.

Example:
  cicd monitor <pipeline-id>
  cicd monitor <run-id>
  cicd monitor <pipeline-id> --interval 5`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		interval, _ := cmd.Flags().GetInt("interval")

		display.PrintInfo(fmt.Sprintf("Monitoring %s (Ctrl+C to stop)", id))

		for {
			// Try to get as pipeline first
			pipeline, pipelineErr := apiClient.GetPipeline(id)
			if pipelineErr == nil {
				// Clear screen and show pipeline status
				fmt.Print("\033[2J\033[H")
				display.PrintPipelineDetails(pipeline)

				if pipeline.Status == "success" || pipeline.Status == "failed" || pipeline.Status == "cancelled" {
					fmt.Printf("\n🏁 Pipeline finished with status: %s\n", pipeline.Status)
					break
				}
			} else {
				// Try to get as run
				run, runErr := apiClient.GetRun(id)
				if runErr == nil {
					// Clear screen and show run status
					fmt.Print("\033[2J\033[H")
					display.PrintRunDetails(run)

					if run.Status == "success" || run.Status == "failed" || run.Status == "cancelled" {
						fmt.Printf("\n🏁 Run finished with status: %s\n", run.Status)
						break
					}
				} else {
					display.PrintError(fmt.Sprintf("ID %s not found as pipeline or run", id))
					display.PrintError(fmt.Sprintf("Pipeline error: %v", pipelineErr))
					display.PrintError(fmt.Sprintf("Run error: %v", runErr))
					return fmt.Errorf("invalid ID: %s", id)
				}
			}

			time.Sleep(time.Duration(interval) * time.Second)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)

	// Add flags
	monitorCmd.Flags().Int("interval", 2, "Refresh interval in seconds")
} 
