package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/xqsit94/shelp/internal/ai"
	"github.com/xqsit94/shelp/internal/config"
	"github.com/xqsit94/shelp/internal/executor"
	"github.com/xqsit94/shelp/internal/history"
	"github.com/xqsit94/shelp/internal/prompt"
)

const defaultHistoryLimit = 20

// runOutcome is what a run ended up doing, which is what gets recorded.
type runOutcome struct {
	commands []string
	executed bool
}

func HistoryCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show recent queries",
		Long:  "List the queries shelp answered, newest first, with the commands it produced.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showHistory(cmd, limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", defaultHistoryLimit, "number of entries to show")

	cmd.AddCommand(historyRunCmd())
	cmd.AddCommand(historyClearCmd())

	return cmd
}

func showHistory(cmd *cobra.Command, limit int) error {
	entries, err := history.Load()
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()

	if len(entries) == 0 {
		fmt.Fprintln(out, "No history yet.")
		return nil
	}

	if limit > len(entries) {
		limit = len(entries)
	}

	width := prompt.GetTerminalWidth()
	now := time.Now()

	for i := 0; i < limit; i++ {
		entry := entries[len(entries)-1-i]

		header := fmt.Sprintf("%s  %s  %s",
			prompt.TitleBoldStyle.Render(strconv.Itoa(i+1)),
			prompt.ExplanationStyle.Render(relativeTime(entry.Time, now)),
			strconv.Quote(entry.Query),
		)
		fmt.Fprintln(out, prompt.Truncate(header, width))

		for j, command := range entry.Commands {
			branch := prompt.TreeBranch
			if j == len(entry.Commands)-1 {
				branch = prompt.TreeLastBranch
			}

			line := fmt.Sprintf("   %s %s %s", prompt.TreeStyle.Render(branch), prompt.Oneline(command), commandStatus(entry, j))
			fmt.Fprintln(out, prompt.Truncate(line, width))
		}
	}

	return nil
}

// Only the exit code of the whole run is recorded, so a failed run marks its
// last command: execution stops there unless the user chose to carry on.
func commandStatus(entry history.Entry, index int) string {
	switch {
	case !entry.Executed:
		return prompt.ExplanationStyle.Render("–")
	case entry.ExitCode != 0 && index == len(entry.Commands)-1:
		return prompt.DangerStyle.Render(fmt.Sprintf("%s (exit %d)", prompt.IconError, entry.ExitCode))
	default:
		return prompt.SuccessStyle.Render(prompt.IconSuccess)
	}
}

func relativeTime(t, now time.Time) string {
	switch elapsed := now.Sub(t); {
	case elapsed < time.Minute:
		return "just now"
	case elapsed < time.Hour:
		return fmt.Sprintf("%dm ago", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(elapsed.Hours()))
	default:
		return t.Format("15:04 02 Jan")
	}
}

func historyRunCmd() *cobra.Command {
	var opts runOptions

	cmd := &cobra.Command{
		Use:   "run [n]",
		Short: "Run the commands of an earlier query again",
		Long:  "Run the commands of entry n again, numbered as in shelp history.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			entry, err := historyEntry(args[0])
			if err != nil {
				return err
			}

			cfg, err := config.LoadProfile(profileName(cmd))
			if err != nil {
				return fmt.Errorf("failed to load configuration: %v", err)
			}

			suggestions := make([]ai.Suggestion, len(entry.Commands))
			for i, command := range entry.Commands {
				suggestions[i] = ai.Suggestion{Command: command}
			}

			var outcome runOutcome
			defer func() { recordHistory(cmd, entry.Query, cfg.Profile, outcome, err) }()

			regenerate, _, err := runSuggestions(cmd, suggestions, entry.Query, executor.DetectShell(), opts, &outcome)
			if regenerate {
				prompt.DisplayWarning("Regenerating is not available for history entries.")
				err = &ExitError{Code: exitCancelled}
			}

			return err
		},
	}

	cmd.Flags().BoolVarP(&opts.print, "print", "p", false, "print the commands instead of running them")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "run the commands without confirmation")
	cmd.Flags().BoolVarP(&opts.copy, "copy", "c", false, "print the commands and copy them to the clipboard")

	return cmd
}

func historyEntry(argument string) (history.Entry, error) {
	number, err := strconv.Atoi(strings.TrimSpace(argument))
	if err != nil || number < 1 {
		return history.Entry{}, &ExitError{Code: 1, Err: fmt.Errorf("invalid history entry %q: pass a number from the list shown by shelp history", argument)}
	}

	entries, err := history.Load()
	if err != nil {
		return history.Entry{}, err
	}

	if number > len(entries) {
		return history.Entry{}, &ExitError{Code: 1, Err: fmt.Errorf("no history entry %d: the history holds %d entries", number, len(entries))}
	}

	entry := entries[len(entries)-number]
	if len(entry.Commands) == 0 {
		return history.Entry{}, &ExitError{Code: 1, Err: fmt.Errorf("history entry %d has no commands", number)}
	}

	return entry, nil
}

func historyClearCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Delete the query history",
		Long:  "Remove the history file.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if !prompt.IsInteractive() {
					return &ExitError{Code: 1, Err: errors.New("history clear needs a terminal to confirm: pass -y")}
				}
				if !prompt.ConfirmYesNoInteractive("Delete the query history?") {
					prompt.DisplayWarning("Clear cancelled.")
					return nil
				}
			}

			if err := history.Clear(); err != nil {
				return err
			}

			prompt.DisplaySuccess("History cleared")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask for confirmation")

	return cmd
}

// recordHistory never fails a run: a history that cannot be written is only
// worth a word under --debug.
func recordHistory(cmd *cobra.Command, query, profile string, outcome runOutcome, err error) {
	if len(outcome.commands) == 0 || historyDisabled(cmd) {
		return
	}

	entry := history.Entry{
		Time:     time.Now(),
		Query:    query,
		Commands: outcome.commands,
		Executed: outcome.executed,
		Profile:  profile,
	}
	if outcome.executed {
		entry.ExitCode = exitCodeOf(err)
	}

	if appendErr := history.Append(entry); appendErr != nil && debugEnabled(cmd) {
		fmt.Fprintf(cmd.ErrOrStderr(), "could not record history: %v\n", appendErr)
	}
}

func historyDisabled(cmd *cobra.Command) bool {
	disabled, _ := cmd.Flags().GetBool("no-history")
	return disabled || os.Getenv("SHELP_NO_HISTORY") == "1"
}

func exitCodeOf(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}

	return 1
}
