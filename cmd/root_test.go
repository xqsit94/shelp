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

// fakeProvider answers with the legacy shape: a plain JSON array of command
// strings.
func fakeProvider(t *testing.T, commands ...string) *httptest.Server {
	t.Helper()

	if commands == nil {
		commands = []string{}
	}

	payload, err := json.Marshal(commands)
	if err != nil {
		t.Fatalf("marshal commands: %v", err)
	}

	return fakeProviderContent(t, string(payload), nil)
}

// fakeProviderContent answers with content verbatim and, when bodies is not
// nil, records the request body sent to it.
func fakeProviderContent(t *testing.T, content string, bodies chan<- map[string]any) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bodies != nil {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode request body: %v", err)
				return
			}
			bodies <- body
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"choices":[{"message":{"content":%s}}]}`, strconv.Quote(content))
	}))
	t.Cleanup(server.Close)

	return server
}

// configureEnv points shelp at the fake provider and clears the remaining
// SHELP_* variables so ambient values cannot leak into a test. It returns the
// scratch config directory.
func configureEnv(t *testing.T, server *httptest.Server) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("SHELP_CONFIG_DIR", dir)
	t.Setenv("SHELP_URL", server.URL)
	t.Setenv("SHELP_API_KEY", "test-key")
	t.Setenv("SHELP_MODEL", "test-model")
	t.Setenv("SHELP_TEMPERATURE", "")
	t.Setenv("SHELP_MAX_TOKENS", "")
	t.Setenv("SHELP_DEBUG", "")
	t.Setenv("SHELP_PROFILE", "")
	t.Setenv("SHELP_NO_HISTORY", "")

	return dir
}

func runRoot(t *testing.T, server *httptest.Server, args ...string) (string, string, error) {
	t.Helper()

	configureEnv(t, server)

	return execRoot(t, args...)
}

func execRoot(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

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

func TestRootPrintModePrintsCommandsOnly(t *testing.T) {
	server := fakeProviderContent(t, `[{"command":"echo hello","explanation":"Prints hello"},{"command":"false","explanation":"Always fails"}]`, nil)

	stdout, stderr, err := runRoot(t, server, "-p", "say", "hello")
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if want := "echo hello\nfalse\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if strings.Contains(stderr, "Prints hello") {
		t.Errorf("stderr = %q, want no explanations", stderr)
	}
}

func TestRootSendsSamplingParameters(t *testing.T) {
	bodies := make(chan map[string]any, 1)
	server := fakeProviderContent(t, `["echo hi"]`, bodies)

	configureEnv(t, server)
	t.Setenv("SHELP_TEMPERATURE", "0.2")
	t.Setenv("SHELP_MAX_TOKENS", "256")

	if _, _, err := execRoot(t, "-p", "say", "hi"); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	body := <-bodies
	if want := 0.2; body["temperature"] != want {
		t.Errorf("temperature = %v, want %v", body["temperature"], want)
	}
	if want := float64(256); body["max_tokens"] != want {
		t.Errorf("max_tokens = %v, want %v", body["max_tokens"], want)
	}
}

func TestRootOmitsUnsetSamplingParameters(t *testing.T) {
	bodies := make(chan map[string]any, 1)
	server := fakeProviderContent(t, `["echo hi"]`, bodies)

	if _, _, err := runRoot(t, server, "-p", "say", "hi"); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	body := <-bodies
	for _, field := range []string{"temperature", "max_tokens"} {
		if _, ok := body[field]; ok {
			t.Errorf("request contains %q, want it omitted", field)
		}
	}
}

// The shell integration hands the whole command line to shelp after --, so a
// query starting with a dash has to reach the provider untouched.
func TestRootSendsLiteralQueryAfterDoubleDash(t *testing.T) {
	bodies := make(chan map[string]any, 1)
	server := fakeProviderContent(t, `["echo hi"]`, bodies)

	stdout, _, err := runRoot(t, server, "-p", "--", "-x list files")
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if stdout != "echo hi\n" {
		t.Errorf("stdout = %q, want %q", stdout, "echo hi\n")
	}

	if got := userMessage(t, <-bodies); got != "-x list files" {
		t.Errorf("query = %q, want %q", got, "-x list files")
	}
}

func userMessage(t *testing.T, body map[string]any) string {
	t.Helper()

	messages, ok := body["messages"].([]any)
	if !ok {
		t.Fatalf("request has no messages: %v", body)
	}

	for _, message := range messages {
		fields, ok := message.(map[string]any)
		if ok && fields["role"] == "user" {
			content, _ := fields["content"].(string)
			return content
		}
	}

	t.Fatalf("request has no user message: %v", body)

	return ""
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
