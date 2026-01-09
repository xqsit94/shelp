package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xqsit94/shelp/internal/ai"
	"github.com/xqsit94/shelp/internal/config"
	"github.com/xqsit94/shelp/internal/executor"
	"github.com/xqsit94/shelp/internal/prompt"
	"github.com/xqsit94/shelp/internal/version"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shelp [query]",
		Short: "Convert natural language to shell commands",
		Long: `shelp - Your AI-powered shell assistant

Convert natural language queries into safe, executable shell commands.
Always prompts for confirmation before execution.

Examples:
  shelp "find all pdf files larger than 10MB"
  shelp "show disk usage for current directory"
  shelp "list all running docker containers"`,
		Version: version.String(),
		Args:    cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			query := strings.Join(args, " ")
			return runQuery(query)
		},
	}

	cmd.AddCommand(ConfigCmd())

	return cmd
}

func runQuery(query string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	if !cfg.IsConfigured() {
		if err := runFirstTimeSetup(cfg); err != nil {
			return err
		}
	}

	shell := executor.DetectShell()
	fmt.Printf("🔍 Generating commands for: %s\n", query)
	fmt.Printf("🐚 Detected shell: %s\n", shell)

	client := ai.NewClient(cfg.AIURL, cfg.APIKey, cfg.Model)
	commands, err := client.GenerateCommands(query, shell)
	if err != nil {
		prompt.DisplayError(fmt.Sprintf("Failed to generate commands: %v", err))
		return nil
	}

	if len(commands) == 0 {
		prompt.DisplayWarning("No commands generated. The request may be unclear or potentially unsafe.")
		return nil
	}

	return executeCommands(commands, shell)
}

func runFirstTimeSetup(cfg *config.Config) error {
	fmt.Println("🚀 Welcome to shelp! Let's set up your AI provider.\n")

	aiURL := prompt.PromptInput("Enter AI API URL (e.g., https://openrouter.ai/api/v1/chat/completions): ")
	if aiURL == "" {
		return fmt.Errorf("AI URL is required")
	}
	cfg.AIURL = aiURL

	apiKey, err := config.PromptForAPIKey()
	if err != nil {
		return err
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	cfg.APIKey = apiKey

	model := prompt.PromptInput("Enter model name (e.g., anthropic/claude-3.5-sonnet): ")
	if model == "" {
		return fmt.Errorf("model name is required")
	}
	cfg.Model = model

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	prompt.DisplaySuccess("Configuration saved!")
	fmt.Println()

	return nil
}

func executeCommands(commands []string, shell string) error {
	total := len(commands)

	for i, cmd := range commands {
		prompt.DisplayCommand(cmd, i+1, total)

		if !prompt.ConfirmExecution(cmd) {
			if i < total-1 {
				if !prompt.ConfirmYesNo("Skip to next command? (y/n): ") {
					fmt.Println("Execution cancelled.")
					return nil
				}
				continue
			}
			fmt.Println("Execution cancelled.")
			return nil
		}

		fmt.Println("\n⏳ Executing...")

		result, err := executor.Execute(cmd, shell)
		if err != nil {
			prompt.DisplayError(fmt.Sprintf("Execution failed: %v", err))
			if i < total-1 {
				if !prompt.ConfirmYesNo("Continue with next command? (y/n): ") {
					return nil
				}
			}
			continue
		}

		if result.Output != "" {
			fmt.Println("\n📤 Output:")
			prompt.DisplayOutput(result.Output, false)
		}

		if result.Error != "" {
			fmt.Println("\n⚠️  Stderr:")
			prompt.DisplayOutput(result.Error, true)
		}

		if result.ExitCode != 0 {
			prompt.DisplayWarning(fmt.Sprintf("Command exited with code %d", result.ExitCode))
		} else {
			prompt.DisplaySuccess("Command completed successfully")
		}
	}

	return nil
}
