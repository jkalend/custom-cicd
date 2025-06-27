package cmd

import (
	"fmt"

	"custom-cicd-cli/internal/client"
	"custom-cicd-cli/internal/config"
	"custom-cicd-cli/internal/display"

	"github.com/spf13/cobra"
)

var (
	cfgFile   string
	apiURL    string
	cfg       *config.Config
	apiClient *client.Client
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cicd",
	Short: "Custom CI/CD Pipeline CLI",
	Long: `A CLI tool for managing CI/CD pipelines and runs.

This CLI connects to your CI/CD backend API to manage pipelines,
monitor runs, and provide real-time status updates.

Example usage:
  cicd pipeline create pipeline.json
  cicd pipeline run <pipeline-id>
  cicd run list
  cicd monitor <pipeline-id>`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error

		// Load configuration
		cfg, err = config.LoadConfig()
		if err != nil {
			display.PrintWarning(fmt.Sprintf("Could not load config: %v", err))
			cfg = config.DefaultConfig()
		}

		// Override with command line flag if provided
		if apiURL != "" {
			cfg.APIURL = apiURL
		}

		// Create API client
		apiClient = client.NewClient(cfg.APIURL)

		// Test connection (but don't fail if it's not available)
		if _, err := apiClient.HealthCheck(); err != nil {
			display.PrintWarning(fmt.Sprintf("Could not connect to API at %s: %v", cfg.APIURL, err))
			display.PrintInfo("Make sure the CI/CD backend is running")
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.custom-cicd/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "CI/CD API URL (default: http://localhost:8000)")

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Custom CI/CD CLI v1.0.0")
		},
	})
}
