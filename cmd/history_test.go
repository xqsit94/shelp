package cmd

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xqsit94/shelp/internal/history"
	"github.com/xqsit94/shelp/pkg/paths"
)

func historyEnv(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv(paths.ConfigDirEnv, dir)
	t.Setenv("SHELP_URL", "")
	t.Setenv("SHELP_API_KEY", "")
	t.Setenv("SHELP_MODEL", "")
	t.Setenv("SHELP_PROFILE", "")
	t.Setenv("SHELP_NO_HISTORY", "")

	return dir
}

func seedHistory(t *testing.T, entries ...history.Entry) {
	t.Helper()

	for _, entry := range entries {
		if err := history.Append(entry); err != nil {
			t.Fatalf("Append() returned error: %v", err)
		}
	}
}

func loadHistory(t *testing.T) []history.Entry {
	t.Helper()

	entries, err := history.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	return entries
}

func TestRootRecordsPrintedQueries(t *testing.T) {
	server := fakeProvider(t, "echo hi", "echo bye")
	configureEnv(t, server)

	if _, _, err := execRoot(t, "-p", "list", "files"); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if _, _, err := execRoot(t, "-p", "say", "hi"); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	entries := loadHistory(t)
	if len(entries) != 2 {
		t.Fatalf("history = %d entries, want 2", len(entries))
	}
	if entries[0].Query != "list files" || entries[1].Query != "say hi" {
		t.Errorf("queries = %q, %q, want them oldest first", entries[0].Query, entries[1].Query)
	}
	if want := "echo hi,echo bye"; strings.Join(entries[0].Commands, ",") != want {
		t.Errorf("commands = %v, want %q", entries[0].Commands, want)
	}
	if entries[0].Executed || entries[0].ExitCode != 0 {
		t.Errorf("entry = %+v, want a non-executed entry", entries[0])
	}
	if entries[0].Profile != "default" {
		t.Errorf("profile = %q, want %q", entries[0].Profile, "default")
	}
}

func TestRootRecordsExecutedExitCode(t *testing.T) {
	server := fakeProvider(t, "false")
	configureEnv(t, server)

	_, _, err := execRoot(t, "-y", "fail", "please")

	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("Execute() error = %v, want exit code 1", err)
	}

	entries := loadHistory(t)
	if len(entries) != 1 {
		t.Fatalf("history = %d entries, want 1", len(entries))
	}
	if !entries[0].Executed || entries[0].ExitCode != 1 {
		t.Errorf("entry = %+v, want an executed entry with exit code 1", entries[0])
	}
}

func TestRootHistoryCanBeDisabled(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  string
	}{
		{"flag", []string{"--no-history", "-p", "say", "hi"}, ""},
		{"env", []string{"-p", "say", "hi"}, "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := fakeProvider(t, "echo hi")
			configureEnv(t, server)
			t.Setenv("SHELP_NO_HISTORY", tt.env)

			if _, _, err := execRoot(t, tt.args...); err != nil {
				t.Fatalf("Execute() returned error: %v", err)
			}

			if entries := loadHistory(t); len(entries) != 0 {
				t.Errorf("history = %v, want it empty", entries)
			}
		})
	}
}

func TestRootRecordsNothingWithoutCommands(t *testing.T) {
	server := fakeProvider(t)
	configureEnv(t, server)

	if _, _, err := execRoot(t, "-p", "do", "something"); err == nil {
		t.Fatal("Execute() returned no error")
	}

	if entries := loadHistory(t); len(entries) != 0 {
		t.Errorf("history = %v, want it empty", entries)
	}
}

func TestHistoryListsNewestFirst(t *testing.T) {
	historyEnv(t)

	now := time.Now()
	seedHistory(t,
		history.Entry{Time: now.Add(-2 * time.Hour), Query: "list files", Commands: []string{"ls -la"}},
		history.Entry{Time: now.Add(-time.Minute), Query: "fail", Commands: []string{"echo hi", "false"}, Executed: true, ExitCode: 1},
	)

	stdout, _, err := execRoot(t, "history")
	if err != nil {
		t.Fatalf("history returned error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("history output = %q, want two entries", stdout)
	}
	if !strings.HasPrefix(lines[0], "1 ") || !strings.Contains(lines[0], `"fail"`) {
		t.Errorf("first line = %q, want the newest entry numbered 1", lines[0])
	}
	if !strings.Contains(lines[1], "echo hi") || !strings.Contains(lines[1], "✓") {
		t.Errorf("line = %q, want a successful command", lines[1])
	}
	if !strings.Contains(lines[2], "✕ (exit 1)") {
		t.Errorf("line = %q, want the failed command", lines[2])
	}
	if !strings.HasPrefix(lines[3], "2 ") || !strings.Contains(lines[3], "2h ago") {
		t.Errorf("line = %q, want the older entry numbered 2", lines[3])
	}
	if !strings.Contains(lines[4], "(not run)") {
		t.Errorf("line = %q, want the command marked as not executed", lines[4])
	}
}

func TestHistoryLimit(t *testing.T) {
	historyEnv(t)

	seedHistory(t,
		history.Entry{Time: time.Now(), Query: "first", Commands: []string{"ls"}},
		history.Entry{Time: time.Now(), Query: "second", Commands: []string{"pwd"}},
	)

	stdout, _, err := execRoot(t, "history", "-n", "1")
	if err != nil {
		t.Fatalf("history returned error: %v", err)
	}
	if strings.Contains(stdout, "first") {
		t.Errorf("stdout = %q, want only the newest entry", stdout)
	}
	if !strings.Contains(stdout, "second") {
		t.Errorf("stdout = %q, want the newest entry", stdout)
	}
}

func TestHistoryEmpty(t *testing.T) {
	historyEnv(t)

	stdout, _, err := execRoot(t, "history")
	if err != nil {
		t.Fatalf("history returned error: %v", err)
	}
	if want := "No history yet.\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

func TestHistoryRunPrintsCommands(t *testing.T) {
	historyEnv(t)

	seedHistory(t, history.Entry{Time: time.Now(), Query: "list files", Commands: []string{"ls -la", "pwd"}})

	stdout, _, err := execRoot(t, "history", "run", "1", "-p")
	if err != nil {
		t.Fatalf("history run returned error: %v", err)
	}
	if want := "ls -la\npwd\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}

	entries := loadHistory(t)
	if len(entries) != 2 {
		t.Fatalf("history = %d entries, want the re-run recorded", len(entries))
	}
	if entries[1].Query != "list files" {
		t.Errorf("query = %q, want the original query", entries[1].Query)
	}
}

func TestHistoryRunRejectsBadEntry(t *testing.T) {
	tests := []struct {
		name     string
		argument string
	}{
		{"not a number", "abc"},
		{"zero", "0"},
		{"out of range", "2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			historyEnv(t)
			seedHistory(t, history.Entry{Time: time.Now(), Query: "list files", Commands: []string{"ls"}})

			_, _, err := execRoot(t, "history", "run", tt.argument)

			var exitErr *ExitError
			if !errors.As(err, &exitErr) || exitErr.Code != 1 {
				t.Fatalf("history run %q error = %v, want exit code 1", tt.argument, err)
			}
		})
	}
}

func TestHistoryClearNeedsConfirmation(t *testing.T) {
	historyEnv(t)
	seedHistory(t, history.Entry{Time: time.Now(), Query: "list files", Commands: []string{"ls"}})

	_, _, err := execRoot(t, "history", "clear")

	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("history clear error = %v, want exit code 1", err)
	}
	if _, err := os.Stat(history.Path()); err != nil {
		t.Errorf("history file removed without confirmation: %v", err)
	}

	if _, _, err := execRoot(t, "history", "clear", "-y"); err != nil {
		t.Fatalf("history clear -y returned error: %v", err)
	}
	if _, err := os.Stat(history.Path()); !os.IsNotExist(err) {
		t.Errorf("history file still present after clear: %v", err)
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, time.March, 4, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name string
		when time.Time
		want string
	}{
		{"seconds", now.Add(-20 * time.Second), "just now"},
		{"minutes", now.Add(-5 * time.Minute), "5m ago"},
		{"hours", now.Add(-3 * time.Hour), "3h ago"},
		{"days", now.Add(-49 * time.Hour), "14:04 02 Mar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := relativeTime(tt.when, now); got != tt.want {
				t.Errorf("relativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}
