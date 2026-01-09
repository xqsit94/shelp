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
	client := ai.NewClient(cfg.AIURL, cfg.APIKey, cfg.Model)

	currentQuery := query

	for {
		spinner := prompt.NewSpinner("Generating commands...")
		spinner.Start()

		commands, err := client.GenerateCommands(currentQuery, shell)

		spinner.Stop()

		if err != nil {
			prompt.DisplayError(fmt.Sprintf("Failed to generate commands: %v", err))
			return nil
		}

		if len(commands) == 0 {
			prompt.DisplayWarning("No commands generated. The request may be unclear or potentially unsafe.")
			return nil
		}

		result := prompt.SelectCommands(commands, currentQuery)

		if result.Cancelled {
			prompt.DisplayWarning("Execution cancelled.")
			return nil
		}

		if result.Regenerate {
			currentQuery = result.NewQuery
			continue
		}

		return executeSelectedCommands(result.SelectedCommands, shell)
	}
}

func runFirstTimeSetup(cfg *config.Config) error {
	result := prompt.RunSetupWizard()

	if result.Cancelled {
		return fmt.Errorf("setup cancelled")
	}

	if result.AIURL == "" {
		return fmt.Errorf("AI URL is required")
	}
	cfg.AIURL = result.AIURL

	if result.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	cfg.APIKey = result.APIKey

	if result.Model == "" {
		return fmt.Errorf("model name is required")
	}
	cfg.Model = result.Model

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	fmt.Println()
	prompt.DisplaySuccess("Configuration saved!")
	fmt.Println()

	return nil
}

func executeSelectedCommands(commands []string, shell string) error {
	if len(commands) == 0 {
		prompt.DisplayWarning("No commands selected.")
		return nil
	}

	total := len(commands)

	for i, cmd := range commands {
		progress := prompt.NewBatchExecutionProgress(i+1, total, cmd)
		progress.Start()

		execResult, err := executor.Execute(cmd, shell)

		progress.Stop()

		cmdPreview := cmd
		if len(cmdPreview) > 50 {
			cmdPreview = cmdPreview[:47] + "..."
		}

		if err != nil {
			prompt.DisplayError(fmt.Sprintf("[%d/%d] %s", i+1, total, cmdPreview))
			prompt.DisplayError(fmt.Sprintf("  Failed: %v", err))
			if i < total-1 {
				if !prompt.ConfirmYesNoInteractive("Continue with next command?") {
					return nil
				}
			}
			continue
		}

		if execResult.Output != "" {
			fmt.Println()
			prompt.DisplayOutputInteractive(execResult.Output, false)
		}

		if execResult.Error != "" {
			fmt.Println()
			prompt.DisplayOutputInteractive(execResult.Error, true)
		}

		if execResult.ExitCode != 0 {
			prompt.DisplayWarning(fmt.Sprintf("[%d/%d] %s - exited with code %d", i+1, total, cmdPreview, execResult.ExitCode))
		} else {
			prompt.DisplaySuccess(fmt.Sprintf("[%d/%d] %s ✓", i+1, total, cmdPreview))
		}
		fmt.Println()
	}

	return nil
}
