package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"custom-cicd-cli/internal/client"
	"custom-cicd-cli/internal/display"

	"github.com/spf13/cobra"
)

// pipelineCmd represents the pipeline command
var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Manage CI/CD pipelines",
	Long:  `Create, run, list, and manage CI/CD pipelines.`,
}

// pipelineCreateCmd represents the pipeline create command
var pipelineCreateCmd = &cobra.Command{
	Use:   "create [pipeline-file]",
	Short: "Create a new pipeline",
	Long: `Create a new pipeline from a JSON configuration file.
Use '-' to read from stdin.

Example:
  cicd pipeline create pipeline.json
  cat pipeline.json | cicd pipeline create -`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var filename string
		if len(args) == 0 {
			filename = "-"
		} else {
			filename = args[0]
		}

		pipeline, err := loadPipelineFromFile(filename)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to load pipeline: %v", err))
			return err
		}

		response, err := apiClient.CreatePipeline(pipeline)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to create pipeline: %v", err))
			return err
		}

		display.PrintSuccess("Pipeline created successfully!")
		fmt.Printf("📋 Pipeline ID: %s\n", response.PipelineID)
		return nil
	},
}

// pipelineListCmd represents the pipeline list command
var pipelineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all pipelines",
	Long:  `List all pipelines with their status and basic information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pipelines, err := apiClient.ListPipelines()
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to list pipelines: %v", err))
			return err
		}

		display.PrintPipelines(pipelines)
		return nil
	},
}

// pipelineStatusCmd represents the pipeline status command
var pipelineStatusCmd = &cobra.Command{
	Use:   "status <pipeline-id>",
	Short: "Get detailed pipeline status",
	Long:  `Get detailed status information about a specific pipeline including step details.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pipelineID := args[0]

		pipeline, err := apiClient.GetPipeline(pipelineID)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to get pipeline status: %v", err))
			return err
		}

		display.PrintPipelineDetails(pipeline)
		return nil
	},
}

// pipelineRunCmd represents the pipeline run command
var pipelineRunCmd = &cobra.Command{
	Use:   "run <pipeline-id>",
	Short: "Run an existing pipeline",
	Long:  `Run an existing pipeline by ID. By default runs in background.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pipelineID := args[0]
		background, _ := cmd.Flags().GetBool("background")

		response, err := apiClient.RunPipeline(pipelineID, background)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to run pipeline: %v", err))
			return err
		}

		display.PrintSuccess("Pipeline started successfully!")
		fmt.Printf("🚀 Run ID: %s\n", response.RunID)
		return nil
	},
}

// pipelineCancelCmd represents the pipeline cancel command
var pipelineCancelCmd = &cobra.Command{
	Use:   "cancel <pipeline-id>",
	Short: "Cancel a running pipeline",
	Long:  `Cancel a running pipeline by ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pipelineID := args[0]

		err := apiClient.CancelPipeline(pipelineID)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to cancel pipeline: %v", err))
			return err
		}

		display.PrintSuccess(fmt.Sprintf("Pipeline %s cancelled", pipelineID))
		return nil
	},
}

// pipelineDeleteCmd represents the pipeline delete command
var pipelineDeleteCmd = &cobra.Command{
	Use:   "delete <pipeline-id>",
	Short: "Delete a pipeline",
	Long:  `Delete a pipeline by ID. Cannot delete running pipelines.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pipelineID := args[0]

		err := apiClient.DeletePipeline(pipelineID)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to delete pipeline: %v", err))
			return err
		}

		display.PrintSuccess(fmt.Sprintf("Pipeline %s deleted", pipelineID))
		return nil
	},
}

// pipelineCreateAndRunCmd represents the pipeline create-and-run command
var pipelineCreateAndRunCmd = &cobra.Command{
	Use:   "create-and-run [pipeline-file]",
	Short: "Create and immediately run a pipeline",
	Long: `Create a new pipeline from a JSON configuration file and immediately run it.
Use '-' to read from stdin.

Example:
  cicd pipeline create-and-run pipeline.json
  cat pipeline.json | cicd pipeline create-and-run -`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var filename string
		if len(args) == 0 {
			filename = "-"
		} else {
			filename = args[0]
		}

		pipeline, err := loadPipelineFromFile(filename)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to load pipeline: %v", err))
			return err
		}

		response, err := apiClient.CreateAndRunPipeline(pipeline)
		if err != nil {
			display.PrintError(fmt.Sprintf("Failed to create and run pipeline: %v", err))
			return err
		}

		display.PrintSuccess("Pipeline created and started successfully!")
		fmt.Printf("📋 Pipeline ID: %s\n", response.PipelineID)
		fmt.Printf("🚀 Run ID: %s\n", response.RunID)
		return nil
	},
}

// pipelineMonitorCmd represents the pipeline monitor command
var pipelineMonitorCmd = &cobra.Command{
	Use:   "monitor <pipeline-id>",
	Short: "Monitor a pipeline in real-time",
	Long:  `Monitor a pipeline's progress in real-time. Updates every 2 seconds by default.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pipelineID := args[0]
		interval, _ := cmd.Flags().GetInt("interval")

		display.PrintInfo(fmt.Sprintf("Monitoring pipeline %s (Ctrl+C to stop)", pipelineID))

		for {
			pipeline, err := apiClient.GetPipeline(pipelineID)
			if err != nil {
				display.PrintError(fmt.Sprintf("Failed to get pipeline status: %v", err))
				return err
			}

			// Clear screen and show status
			fmt.Print("\033[2J\033[H")
			display.PrintPipelineDetails(pipeline)

			if pipeline.Status == "success" || pipeline.Status == "failed" || pipeline.Status == "cancelled" {
				fmt.Printf("\n🏁 Pipeline finished with status: %s\n", pipeline.Status)
				break
			}

			time.Sleep(time.Duration(interval) * time.Second)
		}

		return nil
	},
}

// loadPipelineFromFile loads a pipeline configuration from a file or stdin
func loadPipelineFromFile(filename string) (*client.Pipeline, error) {
	var reader io.Reader

	if filename == "-" {
		display.PrintInfo("Reading pipeline configuration from stdin...")
		reader = os.Stdin
	} else {
		file, err := os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
		}
		defer file.Close()
		reader = file
	}

	var pipeline client.Pipeline
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&pipeline); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline JSON: %w", err)
	}

	return &pipeline, nil
}

func init() {
	rootCmd.AddCommand(pipelineCmd)

	// Add subcommands
	pipelineCmd.AddCommand(pipelineCreateCmd)
	pipelineCmd.AddCommand(pipelineListCmd)
	pipelineCmd.AddCommand(pipelineStatusCmd)
	pipelineCmd.AddCommand(pipelineRunCmd)
	pipelineCmd.AddCommand(pipelineCancelCmd)
	pipelineCmd.AddCommand(pipelineDeleteCmd)
	pipelineCmd.AddCommand(pipelineCreateAndRunCmd)
	pipelineCmd.AddCommand(pipelineMonitorCmd)

	// Add flags
	pipelineRunCmd.Flags().Bool("background", true, "Run pipeline in background")
	pipelineMonitorCmd.Flags().Int("interval", 2, "Refresh interval in seconds")
} 
