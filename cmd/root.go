package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"github.com/xqsit94/shelp/internal/ai"
	"github.com/xqsit94/shelp/internal/config"
	"github.com/xqsit94/shelp/internal/executor"
	"github.com/xqsit94/shelp/internal/prompt"
	"github.com/xqsit94/shelp/internal/safety"
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

type runOptions struct {
	print bool
	yes   bool
	copy  bool
}

func RootCmd() *cobra.Command {
	var opts runOptions

	cmd := &cobra.Command{
		Use:   "shelp [query]",
		Short: "Convert natural language to shell commands",
		Long: `shelp - Your AI-powered shell assistant

Convert natural language queries into safe, executable shell commands.
Always prompts for confirmation before execution.

Without a terminal (piped or captured output) the commands are printed
instead of run, so shelp can be used in $(...) or through a pipe.

Examples:
  shelp "find all pdf files larger than 10MB"
  shelp -p "show disk usage for current directory"
  shelp -y "list all running docker containers"`,
		Version:       version.String(),
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			return runQuery(cmd, strings.Join(args, " "), opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.print, "print", "p", false, "print the generated commands instead of running them")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "run the generated commands without confirmation")
	cmd.Flags().BoolVarP(&opts.copy, "copy", "c", false, "print the generated commands and copy them to the clipboard")
	cmd.PersistentFlags().Bool("debug", false, "print AI requests and responses to stderr")
	cmd.PersistentFlags().String("profile", "", "provider profile to use")
	cmd.PersistentFlags().Bool("no-history", false, "do not record the query in the history")

	cmd.AddCommand(ConfigCmd())
	cmd.AddCommand(HistoryCmd())
	cmd.AddCommand(InitCmd())

	return cmd
}

func runQuery(cmd *cobra.Command, query string, opts runOptions) (err error) {
	ctx := cmd.Context()

	cfg, err := config.LoadProfile(profileName(cmd))
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	if !cfg.IsConfigured() {
		if !prompt.IsInteractive() {
			return &ExitError{Code: 1, Err: errors.New("shelp is not configured: run it once in an interactive terminal, or set SHELP_URL, SHELP_API_KEY and SHELP_MODEL")}
		}
		if err := runFirstTimeSetup(cmd, cfg); err != nil {
			return err
		}
	}

	shell := executor.DetectShell()

	client := ai.NewClient(cfg.AIURL, cfg.APIKey, cfg.Model)
	client.Temperature = cfg.Temperature
	client.MaxTokens = cfg.MaxTokens
	client.Debug = debugEnabled(cmd)

	var outcome runOutcome
	defer func() { recordHistory(cmd, query, cfg.Profile, outcome, err) }()

	request := ai.Request{Query: query, Shell: shell}

	for {
		suggestions, err := generateCommands(ctx, client, request)
		if err != nil {
			return err
		}

		regenerate, refinement, err := runSuggestions(cmd, suggestions, query, shell, opts, &outcome)
		if err != nil || !regenerate {
			return err
		}

		request.History = append(request.History, ai.Turn{Commands: suggestions, Feedback: refinement})
	}
}

// runSuggestions prints or runs one round of suggestions and reports what
// happened in outcome. It returns true when the user asked for another round.
func runSuggestions(cmd *cobra.Command, suggestions []ai.Suggestion, query, shell string, opts runOptions, outcome *runOutcome) (bool, string, error) {
	ctx := cmd.Context()

	// Without a terminal there is nobody to answer the confirmation prompt, so
	// printing the commands is the only useful thing left to do.
	printOnly := opts.print || opts.copy || (!opts.yes && !prompt.IsInteractive())

	switch {
	case printOnly:
		outcome.commands = commandsOf(suggestions)
		return false, "", printCommands(cmd, suggestions, opts.copy)
	case opts.yes:
		return false, "", executeWithoutConfirmation(ctx, suggestions, shell, outcome)
	}

	result := prompt.SelectCommands(promptSuggestions(suggestions), query)

	switch {
	case result.Cancelled:
		outcome.commands = commandsOf(suggestions)
		prompt.DisplayWarning("Execution cancelled.")
		return false, "", &ExitError{Code: exitCancelled}
	case result.Regenerate:
		return true, result.Refinement, nil
	}

	outcome.commands = result.SelectedCommands
	outcome.executed = len(result.SelectedCommands) > 0

	return false, "", executeSelectedCommands(ctx, result.SelectedCommands, shell, false)
}

func remediationHint(err error) string {
	message := err.Error()

	switch {
	case strings.Contains(message, "status 401"), strings.Contains(message, "status 403"):
		return "Check the API key for this provider: shelp config set key"
	case strings.Contains(message, "status 404"):
		return "Check the endpoint and model: shelp config show, then shelp config set url|model"
	case strings.Contains(message, "status 429"):
		return "The provider is rate limiting. Wait a moment and try again."
	case strings.Contains(message, "no such host"), strings.Contains(message, "connection refused"):
		return "Could not reach the provider. Check the URL and your network: shelp config test"
	default:
		return ""
	}
}

func generateCommands(ctx context.Context, client *ai.Client, request ai.Request) ([]ai.Suggestion, error) {
	suggestions, err := prompt.RunWithSpinner(ctx, "Generating commands...", func(ctx context.Context) ([]ai.Suggestion, error) {
		return client.GenerateCommands(ctx, request)
	})

	if err != nil {
		if cancelled(err) {
			prompt.DisplayWarning("Execution cancelled.")
			return nil, &ExitError{Code: exitCancelled}
		}
		prompt.DisplayError(fmt.Sprintf("Failed to generate commands: %v", err))
		if hint := remediationHint(err); hint != "" {
			prompt.DisplayHint(hint)
		}
		return nil, &ExitError{Code: 1}
	}

	if len(suggestions) == 0 {
		prompt.DisplayWarning("No commands generated. The request may be unclear or potentially unsafe.")
		return nil, &ExitError{Code: 1}
	}

	return suggestions, nil
}

func promptSuggestions(suggestions []ai.Suggestion) []prompt.Suggestion {
	items := make([]prompt.Suggestion, len(suggestions))
	for i, suggestion := range suggestions {
		items[i] = prompt.Suggestion(suggestion)
	}
	return items
}

func commandsOf(suggestions []ai.Suggestion) []string {
	commands := make([]string, len(suggestions))
	for i, suggestion := range suggestions {
		commands[i] = suggestion.Command
	}
	return commands
}

// Explanations are deliberately left out: stdout has to stay usable in $(...)
// and pipelines.
func printCommands(cmd *cobra.Command, suggestions []ai.Suggestion, toClipboard bool) error {
	out := cmd.OutOrStdout()
	highlight := prompt.IsTerminalWriter(out)

	commands := make([]string, 0, len(suggestions))
	for _, suggestion := range suggestions {
		command := suggestion.Command
		commands = append(commands, command)

		if safety.IsBlocked(command) {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s blocked for safety reasons, do not run: %s\n", prompt.IconWarning, prompt.Oneline(command))
		}

		if highlight {
			command = prompt.HighlightCommand(command)
		}
		fmt.Fprintln(out, command)
	}

	if toClipboard {
		if err := clipboard.WriteAll(strings.Join(commands, "\n")); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s could not copy to the clipboard: %v\n", prompt.IconWarning, err)
		}
	}

	return nil
}

func executeWithoutConfirmation(ctx context.Context, suggestions []ai.Suggestion, shell string, outcome *runOutcome) error {
	prompt.DisplayCommandPlan(promptSuggestions(suggestions))

	allowed := make([]string, 0, len(suggestions))
	for _, suggestion := range suggestions {
		if safety.IsBlocked(suggestion.Command) {
			prompt.DisplayWarning("Skipping blocked command: " + prompt.Oneline(suggestion.Command))
			continue
		}
		allowed = append(allowed, suggestion.Command)
	}

	if len(allowed) == 0 {
		outcome.commands = commandsOf(suggestions)
		prompt.DisplayError("Every generated command was blocked for safety reasons.")
		return &ExitError{Code: 1}
	}

	outcome.commands = allowed
	outcome.executed = true

	return executeSelectedCommands(ctx, allowed, shell, true)
}

func cancelled(err error) bool {
	return errors.Is(err, prompt.ErrCancelled) || errors.Is(err, context.Canceled)
}

func debugEnabled(cmd *cobra.Command) bool {
	debug, _ := cmd.Flags().GetBool("debug")
	return debug || os.Getenv("SHELP_DEBUG") == "1"
}

func profileName(cmd *cobra.Command) string {
	name, _ := cmd.Flags().GetString("profile")
	return name
}

func runFirstTimeSetup(cmd *cobra.Command, cfg *config.Config) error {
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

	if err := saveProfile(cfg); err != nil {
		return err
	}

	fmt.Println()
	prompt.DisplaySuccess("Configuration saved!")
	warnInsecureURL(cfg.AIURL)

	if err := testConnection(cmd, cfg); err != nil {
		prompt.DisplayWarning("The configuration was saved but the connection test failed.")
		prompt.DisplayWarning("Fix it with: shelp config set url|key|model, then run: shelp config test")
		return err
	}

	fmt.Println()

	return nil
}

// saveProfile writes the wizard answers into the resolved profile, leaving the
// values that came from the environment out of the file.
func saveProfile(cfg *config.Config) error {
	_, err := config.UpdateProfile(cfg.Profile, func(profile *config.Profile) {
		profile.AIURL = cfg.AIURL
		profile.APIKey = cfg.APIKey
		profile.Model = cfg.Model
	})
	if err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	return nil
}

type commandResult struct {
	command     string
	exitCode    int
	interrupted bool
	execErr     error
}

// executeSelectedCommands runs the commands in order. When unattended (--yes)
// a failure stops the run instead of asking whether to carry on, so the whole
// invocation stays free of prompts.
func executeSelectedCommands(ctx context.Context, commands []string, shell string, unattended bool) error {
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

		prompt.DisplayStepResult(result.exitCode, result.interrupted, result.execErr)

		if result.interrupted || ctx.Err() != nil {
			break
		}

		failed := result.execErr != nil || result.exitCode != 0
		if failed && i < total-1 {
			if unattended {
				prompt.DisplayWarning("Stopping: the previous command failed.")
				break
			}
			fmt.Println()
			if !prompt.ConfirmYesNoInteractive("Continue with next command?") {
				break
			}
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
