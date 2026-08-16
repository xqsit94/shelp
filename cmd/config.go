package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"github.com/xqsit94/shelp/internal/ai"
	"github.com/xqsit94/shelp/internal/config"
	"github.com/xqsit94/shelp/internal/executor"
	"github.com/xqsit94/shelp/internal/prompt"
)

const connectionTestQuery = "print the text hello"

func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage shelp configuration",
		Long:  "View and modify shelp configuration settings including AI provider URL, API key, and model.",
	}

	cmd.AddCommand(configSetCmd())
	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configTestCmd())
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
			cfg, err := config.LoadFile()
			if err != nil {
				return err
			}

			cfg.AIURL = args[0]

			if err := config.Save(cfg); err != nil {
				return err
			}

			prompt.DisplaySuccess("AI URL updated successfully")
			warnInsecureURL(cfg.AIURL)

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
			cfg, err := config.LoadFile()
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
			cfg, err := config.LoadFile()
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

			displayConfigTable(
				configValue(cfg.AIURL, cfg.FromEnv.AIURL),
				configValue(cfg.MaskedAPIKey(), cfg.FromEnv.APIKey),
				configValue(cfg.Model, cfg.FromEnv.Model),
			)

			return nil
		},
	}
}

func configValue(value string, fromEnv bool) string {
	if value == "" {
		return "(not set)"
	}
	if fromEnv {
		return value + " (from env)"
	}
	return value
}

func displayConfigTable(aiURL, apiKey, model string) {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(prompt.TableBorderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				return prompt.TableLabelStyle
			}
			return prompt.TableValueStyle
		}).
		Headers("Setting", "Value").
		Row("AI URL", aiURL).
		Row("API Key", apiKey).
		Row("Model", model)

	title := prompt.TitleBoldStyle.
		Foreground(prompt.ColorPrimary).
		Render("Configuration")

	fmt.Println()
	fmt.Println(title)
	fmt.Println(t)
	fmt.Println()
}

func configTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test the AI provider connection",
		Long:  "Send one harmless request to the configured AI provider and report the result.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			return testConnection(cmd, cfg)
		},
	}
}

func testConnection(cmd *cobra.Command, cfg *config.Config) error {
	if !cfg.IsConfigured() {
		prompt.DisplayError("Configuration is incomplete. Set the URL, API key and model first.")
		return &ExitError{Code: 1}
	}

	client := ai.NewClient(cfg.AIURL, cfg.APIKey, cfg.Model)
	client.Debug = debugEnabled(cmd)

	request := ai.Request{Query: connectionTestQuery, Shell: executor.DetectShell()}

	start := time.Now()
	commands, err := prompt.RunWithSpinner(cmd.Context(), "Testing connection...", func(ctx context.Context) ([]string, error) {
		return client.GenerateCommands(ctx, request)
	})
	elapsed := time.Since(start).Round(time.Millisecond)

	if err != nil {
		if cancelled(err) {
			prompt.DisplayWarning("Connection test cancelled.")
			return &ExitError{Code: exitCancelled}
		}
		prompt.DisplayError(err.Error())
		return &ExitError{Code: 1}
	}

	prompt.DisplaySuccess(fmt.Sprintf("Connected to %s as %s — %d command(s) in %s",
		cfg.AIURL, cfg.Model, len(commands), elapsed))

	if len(commands) > 0 {
		fmt.Println(prompt.IndentUnder("  "+prompt.TreeStyle.Render(prompt.TreeLastBranch)+" ", prompt.HighlightCommand(commands[0])))
	}

	return nil
}

func warnInsecureURL(url string) {
	if config.InsecureURL(url) {
		prompt.DisplayWarning("API key will be sent in cleartext over " + url)
	}
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
