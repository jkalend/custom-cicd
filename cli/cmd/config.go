package cmd

import (
	"fmt"

	"custom-cicd-cli/internal/config"
	"custom-cicd-cli/internal/display"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  `View and manage CLI configuration settings.`,
}

// configViewCmd represents the config view command
var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View current configuration",
	Long:  `Display the current CLI configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			cfg = config.DefaultConfig()
			display.PrintWarning(fmt.Sprintf("Using default config due to error: %v", err))
		}

		fmt.Printf("📋 Current Configuration:\n")
		fmt.Printf("  API URL: %s\n", cfg.APIURL)
		return nil
	},
}

// configSetCmd represents the config set command
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value and save it to the config file.

Available keys:
  api-url    Set the CI/CD API URL

Example:
  cicd config set api-url http://localhost:8000`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cfg, err := config.LoadConfig()
		if err != nil {
			cfg = config.DefaultConfig()
			display.PrintWarning("Creating new config file")
		}

		switch key {
		case "api-url":
			cfg.APIURL = value
		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		if err := config.SaveConfig(cfg); err != nil {
			display.PrintError(fmt.Sprintf("Failed to save config: %v", err))
			return err
		}

		display.PrintSuccess(fmt.Sprintf("Configuration updated: %s = %s", key, value))
		return nil
	},
}

// configResetCmd represents the config reset command
var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset configuration to defaults",
	Long:  `Reset all configuration values to their defaults.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.DefaultConfig()

		if err := config.SaveConfig(cfg); err != nil {
			display.PrintError(fmt.Sprintf("Failed to save config: %v", err))
			return err
		}

		display.PrintSuccess("Configuration reset to defaults")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configResetCmd)
} 
