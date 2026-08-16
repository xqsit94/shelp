package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/xqsit94/shelp/internal/ai"
	"github.com/xqsit94/shelp/internal/config"
	"github.com/xqsit94/shelp/internal/executor"
	"github.com/xqsit94/shelp/internal/prompt"
	"github.com/xqsit94/shelp/internal/version"
)

const exitCancelled = 130

// ExitError carries the process exit code. A nil Err means the failure was
// already reported to the user.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

func Run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := RootCmd().ExecuteContext(ctx)
	if err == nil {
		return 0
	}

	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		if exitErr.Err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", exitErr.Err)
		}
		return exitErr.Code
	}

	fmt.Fprintf(os.Stderr, "Error: %v\n", err)

	return 1
}

func RootCmd() *cobra.Command {
	var debug bool

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

			return runQuery(cmd.Context(), strings.Join(args, " "), debug)
		},
	}

	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "print AI requests and responses to stderr")

	cmd.AddCommand(ConfigCmd())

	return cmd
}

func runQuery(ctx context.Context, query string, debug bool) error {
	if !prompt.IsInteractive() {
		return &ExitError{Code: 1, Err: errors.New("shelp needs an interactive terminal")}
	}

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
	client.Debug = debug || os.Getenv("SHELP_DEBUG") == "1"

	request := ai.Request{Query: query, Shell: shell}

	for {
		commands, err := prompt.RunWithSpinner(ctx, "Generating commands...", func(ctx context.Context) ([]string, error) {
			return client.GenerateCommands(ctx, request)
		})

		if err != nil {
			if cancelled(err) {
				prompt.DisplayWarning("Execution cancelled.")
				return &ExitError{Code: exitCancelled}
			}
			prompt.DisplayError(fmt.Sprintf("Failed to generate commands: %v", err))
			return &ExitError{Code: 1}
		}

		if len(commands) == 0 {
			prompt.DisplayWarning("No commands generated. The request may be unclear or potentially unsafe.")
			return &ExitError{Code: 1}
		}

		result := prompt.SelectCommands(commands, query)

		if result.Cancelled {
			prompt.DisplayWarning("Execution cancelled.")
			return &ExitError{Code: exitCancelled}
		}

		if result.Regenerate {
			request.History = append(request.History, ai.Turn{Commands: commands, Feedback: result.Refinement})
			continue
		}

		return executeSelectedCommands(ctx, result.SelectedCommands, shell)
	}
}

func cancelled(err error) bool {
	return errors.Is(err, prompt.ErrCancelled) || errors.Is(err, context.Canceled)
}

func runFirstTimeSetup(cfg *config.Config) error {
	result := prompt.RunSetupWizard()

	if result.Cancelled {
		return &ExitError{Code: exitCancelled, Err: errors.New("setup cancelled")}
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

type commandResult struct {
	command     string
	exitCode    int
	interrupted bool
	execErr     error
}

func executeSelectedCommands(ctx context.Context, commands []string, shell string) error {
	if len(commands) == 0 {
		prompt.DisplayWarning("No commands selected.")
		return nil
	}

	total := len(commands)
	results := make([]commandResult, 0, total)

	for i, command := range commands {
		if ctx.Err() != nil {
			break
		}

		fmt.Println()
		prompt.DisplayRunning(i+1, total, command)

		execResult, err := executor.Execute(ctx, command, shell, executor.Options{})

		result := commandResult{command: command, execErr: err}
		if err == nil {
			result.exitCode = execResult.ExitCode
			result.interrupted = execResult.Interrupted
		}
		results = append(results, result)

		if result.interrupted || ctx.Err() != nil {
			break
		}

		failed := result.execErr != nil || result.exitCode != 0
		if failed && i < total-1 && !prompt.ConfirmYesNoInteractive("Continue with next command?") {
			break
		}
	}

	return summarize(results)
}

func summarize(results []commandResult) error {
	fmt.Println()
	fmt.Println(prompt.TitleBoldStyle.Render(fmt.Sprintf("Executed Commands (%d)", len(results))))

	exitCode := 0

	for i, result := range results {
		branch := prompt.TreeBranch
		if i == len(results)-1 {
			branch = prompt.TreeLastBranch
		}

		styledBranch := prompt.TreeStyle.Render(branch)
		preview := prompt.Truncate(prompt.Oneline(result.command), 50)

		switch {
		case result.execErr != nil:
			fmt.Println(styledBranch + " " + prompt.DangerStyle.Render(preview+" ✕"))
			fmt.Println(prompt.DangerStyle.Render("   Failed: " + result.execErr.Error()))
			exitCode = 1
		case result.interrupted:
			fmt.Println(styledBranch + " " + prompt.DangerStyle.Render(preview+" ✕ (interrupted)"))
			exitCode = exitCancelled
		case result.exitCode != 0:
			fmt.Println(styledBranch + " " + prompt.DangerStyle.Render(fmt.Sprintf("%s ✕ (exit %d)", preview, result.exitCode)))
			exitCode = result.exitCode
		default:
			fmt.Println(styledBranch + " " + prompt.SuccessStyle.Render(preview+" ✓"))
		}
	}

	if exitCode != 0 {
		return &ExitError{Code: exitCode}
	}

	return nil
}
