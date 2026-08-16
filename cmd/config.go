package cmd

import (
	"context"
	"fmt"
	"strconv"
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
	cmd.AddCommand(configUnsetCmd())
	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configTestCmd())
	cmd.AddCommand(configProfileCmd())
	cmd.AddCommand(configResetCmd())

	return cmd
}

func configSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration values",
		Long:  "Set configuration values for AI provider URL, API key, model, or sampling parameters.",
	}

	cmd.AddCommand(configSetURLCmd())
	cmd.AddCommand(configSetKeyCmd())
	cmd.AddCommand(configSetModelCmd())
	cmd.AddCommand(configSetTemperatureCmd())
	cmd.AddCommand(configSetMaxTokensCmd())

	return cmd
}

func configUnsetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset",
		Short: "Clear optional configuration values",
		Long:  "Clear optional configuration values so the provider defaults are used again.",
	}

	cmd.AddCommand(configUnsetValueCmd("temperature", "Temperature", "Clear the sampling temperature", func(profile *config.Profile) {
		profile.Temperature = nil
	}))
	cmd.AddCommand(configUnsetValueCmd("max-tokens", "Max tokens", "Clear the response token limit", func(profile *config.Profile) {
		profile.MaxTokens = nil
	}))

	return cmd
}

func configUnsetValueCmd(name, label, short string, clear func(*config.Profile)) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := config.UpdateProfile(profileName(cmd), clear)
			if err != nil {
				return err
			}

			prompt.DisplaySuccess(fmt.Sprintf("%s cleared in profile %q, the provider default will be used", label, profile))
			return nil
		},
	}
}

func configSetURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "url [url]",
		Short: "Set AI provider URL",
		Long:  "Set the AI API endpoint URL (e.g., https://openrouter.ai/api/v1/chat/completions)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := config.UpdateProfile(profileName(cmd), func(profile *config.Profile) {
				profile.AIURL = args[0]
			})
			if err != nil {
				return err
			}

			prompt.DisplaySuccess(fmt.Sprintf("AI URL updated in profile %q", profile))
			warnInsecureURL(args[0])

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
			apiKey, err := config.PromptForAPIKey()
			if err != nil {
				return err
			}

			if apiKey == "" {
				return fmt.Errorf("API key cannot be empty")
			}

			profile, err := config.UpdateProfile(profileName(cmd), func(profile *config.Profile) {
				profile.APIKey = apiKey
			})
			if err != nil {
				return err
			}

			prompt.DisplaySuccess(fmt.Sprintf("API key updated in profile %q", profile))
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
			profile, err := config.UpdateProfile(profileName(cmd), func(profile *config.Profile) {
				profile.Model = args[0]
			})
			if err != nil {
				return err
			}

			prompt.DisplaySuccess(fmt.Sprintf("Model updated in profile %q", profile))
			return nil
		},
	}
}

func configSetTemperatureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "temperature [value]",
		Short: "Set the sampling temperature",
		Long:  "Set the sampling temperature between 0 and 2. Not sent to the provider unless set.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			temperature, err := config.ParseTemperature(args[0])
			if err != nil {
				return fmt.Errorf("invalid temperature: %v", err)
			}

			profile, err := config.UpdateProfile(profileName(cmd), func(profile *config.Profile) {
				profile.Temperature = &temperature
			})
			if err != nil {
				return err
			}

			prompt.DisplaySuccess(fmt.Sprintf("Temperature updated in profile %q", profile))
			return nil
		},
	}
}

func configSetMaxTokensCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "max-tokens [value]",
		Short: "Set the response token limit",
		Long:  "Set the maximum number of tokens in the response. Not sent to the provider unless set.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			maxTokens, err := config.ParseMaxTokens(args[0])
			if err != nil {
				return fmt.Errorf("invalid max tokens: %v", err)
			}

			profile, err := config.UpdateProfile(profileName(cmd), func(profile *config.Profile) {
				profile.MaxTokens = &maxTokens
			})
			if err != nil {
				return err
			}

			prompt.DisplaySuccess(fmt.Sprintf("Max tokens updated in profile %q", profile))
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
			cfg, err := config.LoadProfile(profileName(cmd))
			if err != nil {
				return err
			}

			if !cfg.IsConfigured() {
				prompt.DisplayWarning("Configuration is incomplete")
			}

			displayConfigTable(
				cfg.Profile,
				configValue(cfg.AIURL, cfg.FromEnv.AIURL),
				configValue(cfg.MaskedAPIKey(), cfg.FromEnv.APIKey),
				configValue(cfg.Model, cfg.FromEnv.Model),
				optionalConfigValue(temperatureValue(cfg), cfg.FromEnv.Temperature),
				optionalConfigValue(maxTokensValue(cfg), cfg.FromEnv.MaxTokens),
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

// Sampling parameters are omitted from the request when unset, so an empty
// value means the provider decides.
func optionalConfigValue(value string, fromEnv bool) string {
	if value == "" {
		return "(provider default)"
	}
	if fromEnv {
		return value + " (from env)"
	}
	return value
}

func temperatureValue(cfg *config.Config) string {
	if cfg.Temperature == nil {
		return ""
	}
	return strconv.FormatFloat(*cfg.Temperature, 'g', -1, 64)
}

func maxTokensValue(cfg *config.Config) string {
	if cfg.MaxTokens == nil {
		return ""
	}
	return strconv.Itoa(*cfg.MaxTokens)
}

func displayConfigTable(profile, aiURL, apiKey, model, temperature, maxTokens string) {
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
		Row("Model", model).
		Row("Temperature", temperature).
		Row("Max tokens", maxTokens)

	title := prompt.TitleBoldStyle.
		Foreground(prompt.ColorPrimary).
		Render(fmt.Sprintf("Configuration (profile: %s)", profile))

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
			cfg, err := config.LoadProfile(profileName(cmd))
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
	client.Temperature = cfg.Temperature
	client.MaxTokens = cfg.MaxTokens
	client.Debug = debugEnabled(cmd)

	request := ai.Request{Query: connectionTestQuery, Shell: executor.DetectShell()}

	start := time.Now()
	suggestions, err := prompt.RunWithSpinner(cmd.Context(), "Testing connection...", func(ctx context.Context) ([]ai.Suggestion, error) {
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
		cfg.AIURL, cfg.Model, len(suggestions), elapsed))

	if len(suggestions) > 0 {
		fmt.Println(prompt.IndentUnder("  "+prompt.TreeStyle.Render(prompt.TreeLastBranch)+" ", prompt.HighlightCommand(suggestions[0].Command)))
		if explanation := suggestions[0].Explanation; explanation != "" {
			fmt.Println(prompt.Truncate("     "+prompt.ExplanationStyle.Render(explanation), prompt.GetTerminalWidth()))
		}
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
		Long:  "Remove all stored configuration settings, including every profile",
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
