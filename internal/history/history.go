// Package history stores the queries shelp answered, one JSON object per line.
package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xqsit94/shelp/pkg/paths"
)

const (
	FileName = "history.jsonl"

	// maxEntries caps the file so an append-only log cannot grow forever.
	maxEntries = 1000

	// maxLineSize allows for long multi-line commands.
	maxLineSize = 1 << 20
)

type Entry struct {
	Time     time.Time `json:"time"`
	Query    string    `json:"query"`
	Commands []string  `json:"commands"`
	Executed bool      `json:"executed"`
	ExitCode int       `json:"exit_code"`
	Profile  string    `json:"profile"`
}

func Path() string {
	return filepath.Join(paths.GetConfigDir(), FileName)
}

func Append(entry Entry) error {
	if err := paths.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to serialize history entry: %v", err)
	}

	file, err := os.OpenFile(Path(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open history file: %v", err)
	}

	if _, err := file.Write(append(line, '\n')); err != nil {
		file.Close()
		return fmt.Errorf("failed to write history file: %v", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to write history file: %v", err)
	}

	return trim()
}

// Load returns every entry, oldest first. Lines that cannot be parsed are
// skipped so a truncated or hand-edited file still lists.
func Load() ([]Entry, error) {
	file, err := os.Open(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read history file: %v", err)
	}
	defer file.Close()

	var entries []Entry

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read history file: %v", err)
	}

	return entries, nil
}

func Clear() error {
	if err := os.Remove(Path()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove history file: %v", err)
	}
	return nil
}

// trim keeps the newest maxEntries lines, rewriting them verbatim so that
// unparsable lines are dropped only by age.
func trim() error {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read history file: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) <= maxEntries {
		return nil
	}

	kept := strings.Join(lines[len(lines)-maxEntries:], "\n") + "\n"
	if err := os.WriteFile(Path(), []byte(kept), 0600); err != nil {
		return fmt.Errorf("failed to write history file: %v", err)
	}

	return nil
}
