package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func fakeProvider(t *testing.T, commands ...string) *httptest.Server {
	t.Helper()

	if commands == nil {
		commands = []string{}
	}

	payload, err := json.Marshal(commands)
	if err != nil {
		t.Fatalf("marshal commands: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"choices":[{"message":{"content":%s}}]}`, strconv.Quote(string(payload)))
	}))
	t.Cleanup(server.Close)

	return server
}

func runRoot(t *testing.T, server *httptest.Server, args ...string) (string, string, error) {
	t.Helper()

	t.Setenv("SHELP_CONFIG_DIR", t.TempDir())
	t.Setenv("SHELP_URL", server.URL)
	t.Setenv("SHELP_API_KEY", "test-key")
	t.Setenv("SHELP_MODEL", "test-model")
	t.Setenv("SHELP_DEBUG", "")

	var stdout, stderr bytes.Buffer

	cmd := RootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	err := cmd.ExecuteContext(t.Context())

	return stdout.String(), stderr.String(), err
}

func TestRootPrintMode(t *testing.T) {
	server := fakeProvider(t, "echo hi")

	stdout, _, err := runRoot(t, server, "-p", "list", "files")
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if stdout != "echo hi\n" {
		t.Errorf("stdout = %q, want %q", stdout, "echo hi\n")
	}
}

func TestRootPrintsWithoutTerminal(t *testing.T) {
	server := fakeProvider(t, "echo hi", "echo bye")

	stdout, _, err := runRoot(t, server, "list", "files")
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if want := "echo hi\necho bye\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

func TestRootPrintModeWarnsAboutBlockedCommands(t *testing.T) {
	server := fakeProvider(t, "rm -rf /")

	stdout, stderr, err := runRoot(t, server, "--print", "wipe", "everything")
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if stdout != "rm -rf /\n" {
		t.Errorf("stdout = %q, want the command printed", stdout)
	}
	if !strings.Contains(stderr, "blocked") {
		t.Errorf("stderr = %q, want a blocked warning", stderr)
	}
}

func TestRootYesRunsCommands(t *testing.T) {
	server := fakeProvider(t, "true")

	if _, _, err := runRoot(t, server, "-y", "do", "nothing"); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
}

func TestRootYesRefusesBlockedCommands(t *testing.T) {
	server := fakeProvider(t, "rm -rf /")

	_, _, err := runRoot(t, server, "--yes", "wipe", "everything")

	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("Execute() error = %v, want exit code 1", err)
	}
}

func TestRootPrintWinsOverYes(t *testing.T) {
	server := fakeProvider(t, "echo hi")

	stdout, _, err := runRoot(t, server, "-p", "-y", "say", "hi")
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if stdout != "echo hi\n" {
		t.Errorf("stdout = %q, want the command printed instead of executed", stdout)
	}
}

func TestRootNoCommandsExitsOne(t *testing.T) {
	server := fakeProvider(t)

	_, _, err := runRoot(t, server, "-p", "delete", "everything")

	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("Execute() error = %v, want exit code 1", err)
	}
}

func TestRootUnconfiguredWithoutTerminal(t *testing.T) {
	t.Setenv("SHELP_CONFIG_DIR", t.TempDir())
	t.Setenv("SHELP_URL", "")
	t.Setenv("SHELP_API_KEY", "")
	t.Setenv("SHELP_MODEL", "")

	cmd := RootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"-p", "list", "files"})

	err := cmd.ExecuteContext(t.Context())

	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("Execute() error = %v, want exit code 1", err)
	}
	if !strings.Contains(err.Error(), "SHELP_URL") {
		t.Errorf("error = %q, want it to mention the environment variables", err)
	}
}
