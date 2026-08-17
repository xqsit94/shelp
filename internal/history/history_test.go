package history

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xqsit94/shelp/pkg/paths"
)

func isolate(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv(paths.ConfigDirEnv, dir)

	return dir
}

func TestPath(t *testing.T) {
	dir := isolate(t)

	if want := filepath.Join(dir, FileName); Path() != want {
		t.Errorf("Path() = %q, want %q", Path(), want)
	}
}

func TestLoadMissingFile(t *testing.T) {
	isolate(t)

	entries, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Load() = %v, want no entries", entries)
	}
}

func TestAppendLoadRoundTrip(t *testing.T) {
	isolate(t)

	want := []Entry{
		{Time: time.Now().Add(-time.Hour).Round(time.Second), Query: "list files", Commands: []string{"ls -la"}, Profile: "default"},
		{Time: time.Now().Round(time.Second), Query: "fail", Commands: []string{"echo hi", "false"}, Executed: true, ExitCode: 1, Profile: "work"},
	}

	for _, entry := range want {
		if err := Append(entry); err != nil {
			t.Fatalf("Append() returned error: %v", err)
		}
	}

	info, err := os.Stat(Path())
	if err != nil {
		t.Fatalf("stat history file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("history file mode = %04o, want 0600", perm)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("Load() = %d entries, want %d", len(got), len(want))
	}

	for i, entry := range got {
		if !entry.Time.Equal(want[i].Time) {
			t.Errorf("entry %d Time = %v, want %v", i, entry.Time, want[i].Time)
		}
		if entry.Query != want[i].Query || entry.Executed != want[i].Executed || entry.ExitCode != want[i].ExitCode || entry.Profile != want[i].Profile {
			t.Errorf("entry %d = %+v, want %+v", i, entry, want[i])
		}
		if strings.Join(entry.Commands, "\n") != strings.Join(want[i].Commands, "\n") {
			t.Errorf("entry %d Commands = %v, want %v", i, entry.Commands, want[i].Commands)
		}
	}
}

func TestLoadSkipsCorruptLines(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"clean", `{"query":"a"}` + "\n" + `{"query":"b"}` + "\n", []string{"a", "b"}},
		{"broken line", `{"query":"a"}` + "\nnot json\n" + `{"query":"b"}` + "\n", []string{"a", "b"}},
		{"blank lines", "\n" + `{"query":"a"}` + "\n\n", []string{"a"}},
		{"truncated tail", `{"query":"a"}` + "\n" + `{"query":"b`, []string{"a"}},
		{"only garbage", "nope\n", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolate(t)
			writeHistory(t, tt.content)

			entries, err := Load()
			if err != nil {
				t.Fatalf("Load() returned error: %v", err)
			}

			queries := make([]string, 0, len(entries))
			for _, entry := range entries {
				queries = append(queries, entry.Query)
			}
			if strings.Join(queries, ",") != strings.Join(tt.want, ",") {
				t.Errorf("Load() queries = %v, want %v", queries, tt.want)
			}
		})
	}
}

func TestAppendTrimsToNewestEntries(t *testing.T) {
	isolate(t)

	var builder strings.Builder
	for i := 0; i < maxEntries+5; i++ {
		builder.WriteString(`{"query":"q` + strconv.Itoa(i) + `"}` + "\n")
	}
	writeHistory(t, builder.String())

	if err := Append(Entry{Query: "newest"}); err != nil {
		t.Fatalf("Append() returned error: %v", err)
	}

	entries, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if len(entries) != maxEntries {
		t.Fatalf("Load() = %d entries, want %d", len(entries), maxEntries)
	}
	if want := "q6"; entries[0].Query != want {
		t.Errorf("oldest kept entry = %q, want %q", entries[0].Query, want)
	}
	if want := "newest"; entries[len(entries)-1].Query != want {
		t.Errorf("newest entry = %q, want %q", entries[len(entries)-1].Query, want)
	}
}

func TestClear(t *testing.T) {
	isolate(t)

	if err := Append(Entry{Query: "list files", Commands: []string{"ls"}}); err != nil {
		t.Fatalf("Append() returned error: %v", err)
	}

	if err := Clear(); err != nil {
		t.Fatalf("Clear() returned error: %v", err)
	}
	if _, err := os.Stat(Path()); !os.IsNotExist(err) {
		t.Errorf("history file still present after Clear(): %v", err)
	}

	if err := Clear(); err != nil {
		t.Errorf("Clear() on missing file returned error: %v", err)
	}

	entries, err := Load()
	if err != nil {
		t.Fatalf("Load() after Clear() returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Load() after Clear() = %v, want no entries", entries)
	}
}

func writeHistory(t *testing.T, content string) {
	t.Helper()

	if err := paths.EnsureConfigDir(); err != nil {
		t.Fatalf("EnsureConfigDir() returned error: %v", err)
	}
	if err := os.WriteFile(Path(), []byte(content), 0600); err != nil {
		t.Fatalf("write history file: %v", err)
	}
}
