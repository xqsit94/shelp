package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xqsit94/shelp/internal/config"
	"github.com/xqsit94/shelp/internal/prompt"
)

func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage shelp configuration",
		Long:  "View and modify shelp configuration settings including AI provider URL, API key, and model.",
	}

	cmd.AddCommand(configSetCmd())
	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configResetCmd())

	return cmd
}

func configSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration values",
		Long:  "Set configuration values for AI provider URL, API key, or model.",
	}

	cmd.AddCommand(configSetURLCmd())
	cmd.AddCommand(configSetKeyCmd())
	cmd.AddCommand(configSetModelCmd())

	return cmd
}

func configSetURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "url [url]",
		Short: "Set AI provider URL",
		Long:  "Set the AI API endpoint URL (e.g., https://openrouter.ai/api/v1/chat/completions)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			cfg.AIURL = args[0]

			if err := config.Save(cfg); err != nil {
				return err
			}

			prompt.DisplaySuccess("AI URL updated successfully")
			return nil
		},
	}
}

func configSetKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "key",
		Short: "Set API key",
		Long:  "Set the API key for authentication (input will be hidden)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			apiKey, err := config.PromptForAPIKey()
			if err != nil {
				return err
			}

			if apiKey == "" {
				return fmt.Errorf("API key cannot be empty")
			}

			cfg.APIKey = apiKey

			if err := config.Save(cfg); err != nil {
				return err
			}

			prompt.DisplaySuccess("API key updated successfully")
			return nil
		},
	}
}

func configSetModelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "model [model]",
		Short: "Set AI model",
		Long:  "Set the AI model to use (e.g., anthropic/claude-3.5-sonnet)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			cfg.Model = args[0]

			if err := config.Save(cfg); err != nil {
				return err
			}

			prompt.DisplaySuccess("Model updated successfully")
			return nil
		},
	}
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  "Display the current shelp configuration (API key will be masked)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if !cfg.IsConfigured() {
				prompt.DisplayWarning("Configuration is incomplete")
				fmt.Println()
			}

			fmt.Println("Current Configuration:")
			fmt.Println("─────────────────────────────────────────")

			if cfg.AIURL != "" {
				fmt.Printf("AI URL:  %s\n", cfg.AIURL)
			} else {
				fmt.Printf("AI URL:  %s(not set)%s\n", "\033[2m", "\033[0m")
			}

			if cfg.APIKey != "" {
				fmt.Printf("API Key: %s\n", cfg.MaskedAPIKey())
			} else {
				fmt.Printf("API Key: %s(not set)%s\n", "\033[2m", "\033[0m")
			}

			if cfg.Model != "" {
				fmt.Printf("Model:   %s\n", cfg.Model)
			} else {
				fmt.Printf("Model:   %s(not set)%s\n", "\033[2m", "\033[0m")
			}

			return nil
		},
	}
}

func configResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset all configuration",
		Long:  "Remove all stored configuration settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !prompt.ConfirmYesNo("Are you sure you want to reset all configuration? (y/n): ") {
				fmt.Println("Reset cancelled.")
				return nil
			}

			if err := config.Reset(); err != nil {
				return err
			}

			prompt.DisplaySuccess("Configuration reset successfully")
			return nil
		},
	}
}
