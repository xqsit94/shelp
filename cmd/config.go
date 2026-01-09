package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
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
			}

			aiURL := cfg.AIURL
			if aiURL == "" {
				aiURL = "(not set)"
			}

			apiKey := cfg.MaskedAPIKey()
			if cfg.APIKey == "" {
				apiKey = "(not set)"
			}

			model := cfg.Model
			if model == "" {
				model = "(not set)"
			}

			displayConfigTable(aiURL, apiKey, model)

			return nil
		},
	}
}

func displayConfigTable(aiURL, apiKey, model string) {
	purple := lipgloss.Color("#874BFD")
	gray := lipgloss.Color("#666666")
	white := lipgloss.Color("#FFFFFF")
	dimWhite := lipgloss.Color("#AAAAAA")

	headerStyle := lipgloss.NewStyle().
		Foreground(purple).
		Bold(true).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().
		Padding(0, 1)

	labelStyle := cellStyle.Foreground(dimWhite)
	valueStyle := cellStyle.Foreground(white)

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(gray)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				return labelStyle
			}
			return valueStyle
		}).
		Headers("Setting", "Value").
		Row("AI URL", aiURL).
		Row("API Key", apiKey).
		Row("Model", model)

	title := lipgloss.NewStyle().
		Foreground(purple).
		Bold(true).
		Render("Configuration")

	fmt.Println()
	fmt.Println(title)
	fmt.Println(t)
	fmt.Println()

	_ = headerStyle
}

func configResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset all configuration",
		Long:  "Remove all stored configuration settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !prompt.ConfirmYesNoInteractive("Are you sure you want to reset all configuration?") {
				prompt.DisplayWarning("Reset cancelled.")
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
