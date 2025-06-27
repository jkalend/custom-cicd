package cmd

import (
	"fmt"

	"custom-cicd-cli/internal/display"

	"github.com/spf13/cobra"
)

// healthCmd represents the health command
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check API health status",
	Long:  `Check the health status of the CI/CD API backend.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		health, err := apiClient.HealthCheck()
		if err != nil {
			display.PrintError(fmt.Sprintf("Health check failed: %v", err))
			return err
		}

		display.PrintSuccess("API is healthy")
		fmt.Printf("🏥 Status: %s\n", health.Status)
		fmt.Printf("🕐 Timestamp: %s\n", health.Timestamp)
		fmt.Printf("🤖 Agent Status: %s\n", health.AgentStatus)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)
} 
