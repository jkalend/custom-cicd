package cmd

import (
	"fmt"

	"custom-cicd-cli/internal/display"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Manage pipeline runs",
	Long:  `List, monitor, and manage individual pipeline runs.`,
}

// runListCmd represents the run list command
var runListCmd = &cobra.Command{
	Use:   "list [pipeline-id]",
	Short: "List all runs or runs for a specific pipeline",
	Long: `List all runs with their status and basic information.
Optionally filter by pipeline ID.

Example:
  cicd run list
  cicd run list <pipeline-id>`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var pipelineID string
		if len(args) > 0 {
			pipelineID = args[0]
		}

		runs, err := apiClient.ListRuns(pipelineID)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to list runs: %v", err))
			return err
		}

		display.PrintRuns(runs)
		return nil
	},
}

// runStatusCmd represents the run status command
var runStatusCmd = &cobra.Command{
	Use:   "status <run-id>",
	Short: "Get detailed run status",
	Long:  `Get detailed status information about a specific run including step details.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]

		run, err := apiClient.GetRun(runID)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to get run status: %v", err))
			return err
		}

		display.PrintRunDetails(run)
		return nil
	},
}

// runCancelCmd represents the run cancel command
var runCancelCmd = &cobra.Command{
	Use:   "cancel <run-id>",
	Short: "Cancel a running pipeline run",
	Long:  `Cancel a specific running pipeline run by ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]

		err := apiClient.CancelRun(runID)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to cancel run: %v", err))
			return err
		}

		display.PrintSuccess(fmt.Sprintf("Run %s cancelled", runID))
		return nil
	},
}

// runDeleteCmd represents the run delete command
var runDeleteCmd = &cobra.Command{
	Use:   "delete <run-id>",
	Short: "Delete a run",
	Long:  `Delete a specific run by ID. Cannot delete currently running runs.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID := args[0]

		err := apiClient.DeleteRun(runID)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to delete run: %v", err))
			return err
		}

		display.PrintSuccess(fmt.Sprintf("Run %s deleted", runID))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Add subcommands
	runCmd.AddCommand(runListCmd)
	runCmd.AddCommand(runStatusCmd)
	runCmd.AddCommand(runCancelCmd)
	runCmd.AddCommand(runDeleteCmd)
} 
