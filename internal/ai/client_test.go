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

func TestParseSuggestions(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []Suggestion
		wantErr bool
	}{
		{
			name:    "plain array",
			content: `["ls -la", "pwd"]`,
			want:    []Suggestion{{Command: "ls -la"}, {Command: "pwd"}},
		},
		{
			name:    "json fenced",
			content: "```json\n[\"ls -la\"]\n```",
			want:    []Suggestion{{Command: "ls -la"}},
		},
		{
			name:    "bare fenced",
			content: "```\n[\"ls -la\"]\n```",
			want:    []Suggestion{{Command: "ls -la"}},
		},
		{
			name:    "surrounding whitespace",
			content: "  \n\t[\"ls -la\"]\n  ",
			want:    []Suggestion{{Command: "ls -la"}},
		},
		{
			name:    "empty array",
			content: "[]",
			want:    []Suggestion{},
		},
		{
			name:    "embedded in prose",
			content: "Sure! Here you go:\n[\"ls -la\", \"pwd\"]\nHope that helps.",
			want:    []Suggestion{{Command: "ls -la"}, {Command: "pwd"}},
		},
		{
			name:    "commands object",
			content: `{"commands": ["ls -la", "pwd"]}`,
			want:    []Suggestion{{Command: "ls -la"}, {Command: "pwd"}},
		},
		{
			name:    "array of command objects",
			content: `[{"command": "ls -la"}, {"command": "pwd"}]`,
			want:    []Suggestion{{Command: "ls -la"}, {Command: "pwd"}},
		},
		{
			name:    "array of objects with explanations",
			content: `[{"command": "ls -la", "explanation": "Lists files including hidden ones"}]`,
			want:    []Suggestion{{Command: "ls -la", Explanation: "Lists files including hidden ones"}},
		},
		{
			name:    "mixed object and string entries",
			content: `[{"command": "ls -la", "explanation": "Lists files"}, "pwd"]`,
			want:    []Suggestion{{Command: "ls -la", Explanation: "Lists files"}, {Command: "pwd"}},
		},
		{
			name:    "commands object with explanations",
			content: `{"commands": [{"command": "pwd", "explanation": "Prints the working directory"}]}`,
			want:    []Suggestion{{Command: "pwd", Explanation: "Prints the working directory"}},
		},
		{
			name:    "objects embedded in prose",
			content: "Here you go:\n[{\"command\": \"pwd\", \"explanation\": \"Prints the working directory\"}]\nEnjoy.",
			want:    []Suggestion{{Command: "pwd", Explanation: "Prints the working directory"}},
		},
		{
			name:    "strips ansi escapes",
			content: `["\u001b[31mls -la\u001b[0m"]`,
			want:    []Suggestion{{Command: "ls -la"}},
		},
		{
			name:    "drops control characters",
			content: `["ls -la\u0007\r"]`,
			want:    []Suggestion{{Command: "ls -la"}},
		},
		{
			name:    "keeps tabs and newlines",
			content: `["for f in *; do\n\techo $f\ndone"]`,
			want:    []Suggestion{{Command: "for f in *; do\n\techo $f\ndone"}},
		},
		{
			name:    "drops empty entries",
			content: `["ls -la", "", "   ", "pwd"]`,
			want:    []Suggestion{{Command: "ls -la"}, {Command: "pwd"}},
		},
		{
			name:    "sanitizes the explanation",
			content: `[{"command": "ls", "explanation": "\u001b[31mLists\u0007 files\non one line\u0009"}]`,
			want:    []Suggestion{{Command: "ls", Explanation: "Lists files on one line"}},
		},
		{
			name:    "drops the explanation of an empty command",
			content: `[{"command": "   ", "explanation": "Does nothing"}, {"command": "pwd"}]`,
			want:    []Suggestion{{Command: "pwd"}},
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
			got, err := parseSuggestions(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseSuggestions(%q) = %v, want error", tt.content, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSuggestions(%q) returned error: %v", tt.content, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSuggestions(%q) = %#v, want %#v", tt.content, got, tt.want)
			}
		})
	}
}

func TestParseSuggestionsCapsLength(t *testing.T) {
	payload, err := json.Marshal([]Suggestion{{
		Command:     strings.Repeat("a", maxCommandRunes+100),
		Explanation: strings.Repeat("b", maxExplanationRunes+100),
	}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := parseSuggestions(string(payload))
	if err != nil {
		t.Fatalf("parseSuggestions returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("parseSuggestions returned %d commands, want 1", len(got))
	}
	if n := len([]rune(got[0].Command)); n != maxCommandRunes {
		t.Errorf("command length = %d, want %d", n, maxCommandRunes)
	}
	if n := len([]rune(got[0].Explanation)); n != maxExplanationRunes {
		t.Errorf("explanation length = %d, want %d", n, maxExplanationRunes)
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
	if !reflect.DeepEqual(got, []Suggestion{{Command: "ls -la"}}) {
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
	if !reflect.DeepEqual(got, []Suggestion{{Command: "ls -la"}}) {
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
	if !reflect.DeepEqual(got, []Suggestion{{Command: "ls -la"}}) {
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
		{Commands: []Suggestion{{Command: "ls", Explanation: "Lists files"}}, Feedback: "use long format"},
		{Commands: []Suggestion{{Command: "ls -l"}}},
	}

	if _, err := NewClient(server.URL, "key", "model").GenerateCommands(t.Context(), request); err != nil {
		t.Fatalf("GenerateCommands returned error: %v", err)
	}

	got := <-received

	want := []Message{
		{Role: "system"},
		{Role: "user", Content: "list files"},
		{Role: "assistant", Content: `[{"command":"ls","explanation":"Lists files"}]`},
		{Role: "user", Content: "The user rejected those commands. use long format"},
		{Role: "assistant", Content: `[{"command":"ls -l"}]`},
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

func requestBody(t *testing.T) (*httptest.Server, chan map[string]any) {
	t.Helper()

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
	t.Cleanup(server.Close)

	return server, received
}

func TestGenerateCommandsRequestShape(t *testing.T) {
	server, received := requestBody(t)

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

func TestGenerateCommandsSendsSamplingParameters(t *testing.T) {
	server, received := requestBody(t)

	client := NewClient(server.URL, "key", "model")
	temperature, maxTokens := 0.2, 256
	client.Temperature = &temperature
	client.MaxTokens = &maxTokens

	if _, err := client.GenerateCommands(t.Context(), testRequest()); err != nil {
		t.Fatalf("GenerateCommands returned error: %v", err)
	}

	got := <-received

	if want := 0.2; got["temperature"] != want {
		t.Errorf("temperature = %v, want %v", got["temperature"], want)
	}
	if want := float64(256); got["max_tokens"] != want {
		t.Errorf("max_tokens = %v, want %v", got["max_tokens"], want)
	}
}
