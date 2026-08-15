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
		Version:       version.String(),
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
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
	results := make([]struct {
		cmd       string
		success   bool
		exitCode  int
		output    string
		errOutput string
		execErr   error
	}, total)

	for i, cmd := range commands {
		progress := prompt.NewBatchExecutionProgress(i+1, total, cmd)
		progress.Start()

		execResult, err := executor.Execute(cmd, shell)

		progress.Stop()

		results[i].cmd = cmd
		results[i].execErr = err

		if err != nil {
			results[i].success = false
			if i < total-1 {
				if !prompt.ConfirmYesNoInteractive("Continue with next command?") {
					results = results[:i+1]
					break
				}
			}
			continue
		}

		results[i].success = execResult.ExitCode == 0
		results[i].exitCode = execResult.ExitCode
		results[i].output = execResult.Output
		results[i].errOutput = execResult.Error
	}

	fmt.Println()
	fmt.Println(prompt.TitleBoldStyle.Render(fmt.Sprintf("Executed Commands (%d)", len(results))))

	for i, result := range results {
		isLast := i == len(results)-1
		branch := prompt.TreeBranch
		if isLast {
			branch = prompt.TreeLastBranch
		}

		cmdPreview := prompt.Truncate(result.cmd, 50)

		styledBranch := prompt.TreeStyle.Render(branch)

		if result.execErr != nil {
			fmt.Println(styledBranch + " " + prompt.DangerStyle.Render(cmdPreview+" ✕"))
			fmt.Println(prompt.DangerStyle.Render("   Failed: " + result.execErr.Error()))
		} else if !result.success {
			fmt.Println(styledBranch + " " + prompt.DangerStyle.Render(fmt.Sprintf("%s ✕ (exit %d)", cmdPreview, result.exitCode)))
		} else {
			fmt.Println(styledBranch + " " + prompt.SuccessStyle.Render(cmdPreview+" ✓"))
		}

		if result.output != "" {
			fmt.Println()
			prompt.DisplayOutputInteractive(result.output, false)
		}

		if result.errOutput != "" {
			fmt.Println()
			prompt.DisplayOutputInteractive(result.errOutput, true)
		}
	}

	return nil
}
