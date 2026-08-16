package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseCommands(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
		wantErr bool
	}{
		{
			name:    "plain array",
			content: `["ls -la", "pwd"]`,
			want:    []string{"ls -la", "pwd"},
		},
		{
			name:    "json fenced",
			content: "```json\n[\"ls -la\"]\n```",
			want:    []string{"ls -la"},
		},
		{
			name:    "bare fenced",
			content: "```\n[\"ls -la\"]\n```",
			want:    []string{"ls -la"},
		},
		{
			name:    "surrounding whitespace",
			content: "  \n\t[\"ls -la\"]\n  ",
			want:    []string{"ls -la"},
		},
		{
			name:    "empty array",
			content: "[]",
			want:    []string{},
		},
		{
			name:    "embedded in prose",
			content: "Sure! Here you go:\n[\"ls -la\", \"pwd\"]\nHope that helps.",
			want:    []string{"ls -la", "pwd"},
		},
		{
			name:    "commands object",
			content: `{"commands": ["ls -la", "pwd"]}`,
			want:    []string{"ls -la", "pwd"},
		},
		{
			name:    "array of command objects",
			content: `[{"command": "ls -la"}, {"command": "pwd"}]`,
			want:    []string{"ls -la", "pwd"},
		},
		{
			name:    "strips ansi escapes",
			content: `["\u001b[31mls -la\u001b[0m"]`,
			want:    []string{"ls -la"},
		},
		{
			name:    "drops control characters",
			content: `["ls -la\u0007\r"]`,
			want:    []string{"ls -la"},
		},
		{
			name:    "keeps tabs and newlines",
			content: `["for f in *; do\n\techo $f\ndone"]`,
			want:    []string{"for f in *; do\n\techo $f\ndone"},
		},
		{
			name:    "drops empty entries",
			content: `["ls -la", "", "   ", "pwd"]`,
			want:    []string{"ls -la", "pwd"},
		},
		{
			name:    "invalid json",
			content: "not json at all",
			wantErr: true,
		},
		{
			name:    "object instead of array",
			content: `{"command": "ls"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCommands(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseCommands(%q) = %v, want error", tt.content, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCommands(%q) returned error: %v", tt.content, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCommands(%q) = %#v, want %#v", tt.content, got, tt.want)
			}
		})
	}
}

func TestParseCommandsCapsLength(t *testing.T) {
	long := strings.Repeat("a", maxCommandRunes+100)

	payload, err := json.Marshal([]string{long})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := parseCommands(string(payload))
	if err != nil {
		t.Fatalf("parseCommands returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("parseCommands returned %d commands, want 1", len(got))
	}
	if len([]rune(got[0])) != maxCommandRunes {
		t.Errorf("command length = %d, want %d", len([]rune(got[0])), maxCommandRunes)
	}
}

func chatResponse(t *testing.T, content string) string {
	t.Helper()

	encoded, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return fmt.Sprintf(`{"choices":[{"message":{"content":%s}}]}`, encoded)
}

func testRequest() Request {
	return Request{Query: "list files", Shell: "bash"}
}

func TestGenerateCommands(t *testing.T) {
	var requests atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, chatResponse(t, `["ls -la"]`))
	}))
	defer server.Close()

	got, err := NewClient(server.URL, "key", "model").GenerateCommands(t.Context(), testRequest())
	if err != nil {
		t.Fatalf("GenerateCommands returned error: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"ls -la"}) {
		t.Errorf("GenerateCommands = %#v, want [ls -la]", got)
	}
	if n := requests.Load(); n != 1 {
		t.Errorf("made %d requests, want 1", n)
	}
}

func TestGenerateCommandsRetriesServerError(t *testing.T) {
	var requests atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, chatResponse(t, `["ls -la"]`))
	}))
	defer server.Close()

	got, err := NewClient(server.URL, "key", "model").GenerateCommands(t.Context(), testRequest())
	if err != nil {
		t.Fatalf("GenerateCommands returned error: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"ls -la"}) {
		t.Errorf("GenerateCommands = %#v, want [ls -la]", got)
	}
	if n := requests.Load(); n != 2 {
		t.Errorf("made %d requests, want 2", n)
	}
}

func TestGenerateCommandsRetriesRateLimit(t *testing.T) {
	var requests atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, chatResponse(t, `["ls -la"]`))
	}))
	defer server.Close()

	start := time.Now()

	got, err := NewClient(server.URL, "key", "model").GenerateCommands(t.Context(), testRequest())
	if err != nil {
		t.Fatalf("GenerateCommands returned error: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"ls -la"}) {
		t.Errorf("GenerateCommands = %#v, want [ls -la]", got)
	}
	if n := requests.Load(); n != 2 {
		t.Errorf("made %d requests, want 2", n)
	}
	if elapsed := time.Since(start); elapsed > retryBackoff[0] {
		t.Errorf("took %s, want Retry-After: 0 to skip the backoff", elapsed)
	}
}

func TestGenerateCommandsClientError(t *testing.T) {
	var requests atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"invalid api key"}}`)
	}))
	defer server.Close()

	_, err := NewClient(server.URL, "key", "model").GenerateCommands(t.Context(), testRequest())
	if err == nil {
		t.Fatal("GenerateCommands returned no error")
	}
	if want := "API error (status 401): invalid api key"; err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if n := requests.Load(); n != 1 {
		t.Errorf("made %d requests, want 1", n)
	}
}

func TestGenerateCommandsStringErrorField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"model not found"}`)
	}))
	defer server.Close()

	_, err := NewClient(server.URL, "key", "model").GenerateCommands(t.Context(), testRequest())
	if err == nil {
		t.Fatal("GenerateCommands returned no error")
	}
	if want := "API error (status 400): model not found"; err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

func TestGenerateCommandsContextCancelled(t *testing.T) {
	var requests atomic.Int64

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		<-release
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	time.AfterFunc(50*time.Millisecond, cancel)

	start := time.Now()

	_, err := NewClient(server.URL, "key", "model").GenerateCommands(ctx, testRequest())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GenerateCommands error = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("took %s, want a prompt return", elapsed)
	}
	if n := requests.Load(); n != 1 {
		t.Errorf("made %d requests, want 1", n)
	}
}

func TestGenerateCommandsSendsHistory(t *testing.T) {
	received := make(chan ChatRequest, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			return
		}

		var parsed ChatRequest
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Errorf("unmarshal body: %v", err)
			return
		}
		received <- parsed

		fmt.Fprint(w, chatResponse(t, `["ls -la"]`))
	}))
	defer server.Close()

	request := testRequest()
	request.History = []Turn{
		{Commands: []string{"ls"}, Feedback: "use long format"},
		{Commands: []string{"ls -l"}},
	}

	if _, err := NewClient(server.URL, "key", "model").GenerateCommands(t.Context(), request); err != nil {
		t.Fatalf("GenerateCommands returned error: %v", err)
	}

	got := <-received

	want := []Message{
		{Role: "system"},
		{Role: "user", Content: "list files"},
		{Role: "assistant", Content: `["ls"]`},
		{Role: "user", Content: "The user rejected those commands. use long format"},
		{Role: "assistant", Content: `["ls -l"]`},
		{Role: "user", Content: "The user rejected those commands. Propose a different approach."},
	}

	if len(got.Messages) != len(want) {
		t.Fatalf("got %d messages, want %d: %#v", len(got.Messages), len(want), got.Messages)
	}

	for i, message := range want {
		if got.Messages[i].Role != message.Role {
			t.Errorf("message %d role = %q, want %q", i, got.Messages[i].Role, message.Role)
		}
		if i > 0 && got.Messages[i].Content != message.Content {
			t.Errorf("message %d content = %q, want %q", i, got.Messages[i].Content, message.Content)
		}
	}

	if !strings.Contains(got.Messages[0].Content, "bash") {
		t.Errorf("system prompt does not mention the shell: %q", got.Messages[0].Content)
	}
}

func TestGenerateCommandsRequestShape(t *testing.T) {
	received := make(chan map[string]any, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var parsed map[string]any
		if err := json.NewDecoder(r.Body).Decode(&parsed); err != nil {
			t.Errorf("decode body: %v", err)
			return
		}
		received <- parsed

		fmt.Fprint(w, chatResponse(t, `["ls -la"]`))
	}))
	defer server.Close()

	if _, err := NewClient(server.URL, "key", "model").GenerateCommands(t.Context(), testRequest()); err != nil {
		t.Fatalf("GenerateCommands returned error: %v", err)
	}

	got := <-received

	for _, field := range []string{"temperature", "max_tokens", "response_format"} {
		if _, ok := got[field]; ok {
			t.Errorf("request contains %q, want it omitted", field)
		}
	}
}
