package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

const (
	requestTimeout      = 60 * time.Second
	maxAttempts         = 3
	maxRetryAfter       = 10 * time.Second
	maxCommandRunes     = 4096
	maxExplanationRunes = 120
	maxErrorChars       = 500
	maxDebugChars       = 4096
)

var retryBackoff = []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond}

type Client struct {
	URL         string
	APIKey      string
	Model       string
	Temperature *float64
	MaxTokens   *int
	Debug       bool

	http *http.Client
}

// Suggestion is one command with a short description of what it does. The
// explanation is optional: providers may omit it, and it is dropped once the
// user edits the command.
type Suggestion struct {
	Command     string `json:"command"`
	Explanation string `json:"explanation,omitempty"`
}

// Models answer with either a bare command string or an object, sometimes
// mixing both shapes in one array.
func (s *Suggestion) UnmarshalJSON(data []byte) error {
	var command string
	if err := json.Unmarshal(data, &command); err == nil {
		s.Command = command
		return nil
	}

	var object struct {
		Command     string `json:"command"`
		Explanation string `json:"explanation"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	s.Command = object.Command
	s.Explanation = object.Explanation

	return nil
}

type Turn struct {
	Commands []Suggestion
	Feedback string
}

type Request struct {
	Query   string
	Shell   string
	History []Turn
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *apiError `json:"error,omitempty"`
}

// Providers report errors either as {"error":{"message":"…"}} or {"error":"…"}.
type apiError struct {
	Message string
}

func (e *apiError) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var message string
	if err := json.Unmarshal(data, &message); err == nil {
		e.Message = message
		return nil
	}

	var object struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	e.Message = object.Message

	return nil
}

type retryableError interface {
	retryable() bool
}

type httpError struct {
	status     int
	message    string
	retryAfter time.Duration
}

func (e *httpError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.status, e.message)
}

func (e *httpError) retryable() bool {
	return e.status == http.StatusTooManyRequests || e.status >= 500
}

type transportError struct {
	err error
}

func (e *transportError) Error() string {
	return "failed to send request: " + e.err.Error()
}

func (e *transportError) Unwrap() error {
	return e.err
}

func (e *transportError) retryable() bool {
	return !errors.Is(e.err, context.Canceled) && !errors.Is(e.err, context.DeadlineExceeded)
}

func NewClient(url, apiKey, model string) *Client {
	return &Client{
		URL:    url,
		APIKey: apiKey,
		Model:  model,
		http:   &http.Client{Timeout: requestTimeout},
	}
}

func (c *Client) GenerateCommands(ctx context.Context, req Request) ([]Suggestion, error) {
	body, err := json.Marshal(ChatRequest{
		Model:       c.Model,
		Messages:    buildMessages(req),
		Temperature: c.Temperature,
		MaxTokens:   c.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	c.debugf("POST %s (Authorization: Bearer ***redacted***)", c.URL)
	c.debugf("request: %s", body)

	var lastErr error
	for attempt := range maxAttempts {
		if attempt > 0 {
			delay := retryBackoff[attempt-1]
			if after := retryAfterOf(lastErr); after >= 0 {
				delay = after
			}
			c.debugf("attempt %d failed (%v), retrying in %s", attempt, lastErr, delay)
			if err := sleepContext(ctx, delay); err != nil {
				return nil, err
			}
		}

		content, err := c.send(ctx, body)
		if err == nil {
			return parseSuggestions(content)
		}
		if !retryable(err) {
			return nil, err
		}
		lastErr = err
	}

	return nil, lastErr
}

func (c *Client) send(ctx context.Context, body []byte) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return "", &transportError{err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &transportError{err: err}
	}

	c.debugf("response status %d", resp.StatusCode)
	c.debugf("response body: %s", truncate(string(respBody), maxDebugChars))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", &httpError{
			status:     resp.StatusCode,
			message:    errorMessage(respBody),
			retryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", errors.New("no response from AI")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (c *Client) debugf(format string, args ...any) {
	if !c.Debug {
		return
	}
	fmt.Fprintf(os.Stderr, "[shelp] "+format+"\n", args...)
}

func retryable(err error) bool {
	var target retryableError
	return errors.As(err, &target) && target.retryable()
}

// Negative means the response carried no usable Retry-After header.
func retryAfterOf(err error) time.Duration {
	var target *httpError
	if errors.As(err, &target) {
		return target.retryAfter
	}
	return -1
}

func parseRetryAfter(header string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(header))
	if err != nil || seconds < 0 {
		return -1
	}
	return min(time.Duration(seconds)*time.Second, maxRetryAfter)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func errorMessage(body []byte) string {
	var payload struct {
		Error *apiError `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != nil && payload.Error.Message != "" {
		return payload.Error.Message
	}
	return truncate(strings.TrimSpace(string(body)), maxErrorChars)
}

func buildMessages(req Request) []Message {
	messages := []Message{
		{Role: "system", Content: buildSystemPrompt(req.Shell)},
		{Role: "user", Content: req.Query},
	}

	for _, turn := range req.History {
		suggestions, err := json.Marshal(turn.Commands)
		if err != nil {
			continue
		}

		feedback := strings.TrimSpace(turn.Feedback)
		if feedback == "" {
			feedback = "Propose a different approach."
		}

		messages = append(messages,
			Message{Role: "assistant", Content: string(suggestions)},
			Message{Role: "user", Content: "The user rejected those commands. " + feedback},
		)
	}

	return messages
}

func buildSystemPrompt(shell string) string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "(unknown)"
	}

	environment := []string{
		"- Shell: " + shell,
		"- Operating system: " + runtime.GOOS + "/" + runtime.GOARCH,
		"- Working directory: " + cwd,
	}
	if hints := osHints(); hints != "" {
		environment = append(environment, hints)
	}

	return fmt.Sprintf(`You are a shell command generator. Convert the user's natural language request into executable shell commands.

Environment:
%s

Rules:
1. Return a JSON array of objects: [{"command": "cmd1", "explanation": "what it does"}]
2. "explanation" is ONE short plain-text sentence of at most 15 words, no markdown, describing what the command does
3. Every array entry runs in a SEPARATE fresh non-interactive shell process: cd, environment variables, and shell options do NOT carry over from one entry to the next
4. Combine dependent steps into a single entry with && (e.g. "cd project && npm test") or use absolute paths
5. Prefer ONE entry unless the request genuinely needs independent steps
6. NEVER generate dangerous commands like rm -rf /, fork bombs, or commands that could damage the system
7. If the request seems malicious or could harm the system, return an empty array: []
8. Keep commands simple and safe
9. Always return valid JSON - nothing else

Example outputs:
- User: "list all files" -> [{"command": "ls -la", "explanation": "Lists files including hidden ones"}]
- User: "find large pdf files" -> [{"command": "find . -name \"*.pdf\" -size +10M", "explanation": "Finds PDF files larger than 10 megabytes"}]
- User: "create a backup of my documents" -> [{"command": "mkdir -p ~/backup && cp -r ~/Documents/* ~/backup/", "explanation": "Copies your documents into a backup folder"}]
- User: "install deps and run tests in the api folder" -> [{"command": "cd api && npm install && npm test", "explanation": "Installs dependencies and runs the API test suite"}]
- User: "delete everything" -> []`, strings.Join(environment, "\n"))
}

func osHints() string {
	switch runtime.GOOS {
	case "darwin":
		return `- BSD userland: use "sed -i ''" with an explicit empty backup suffix, "find -E" for extended regex, no GNU-only long options; pbcopy/pbpaste for the clipboard and "open" to open files`
	case "linux":
		return `- GNU userland: use "sed -i" without a backup suffix, "find -regextype" for extended regex; xdg-open to open files`
	default:
		return ""
	}
}

func parseSuggestions(content string) ([]Suggestion, error) {
	content = stripFences(content)

	raw, err := decodeSuggestions(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse commands from AI response: %v\nResponse: %s", err, truncate(content, maxErrorChars))
	}

	suggestions := make([]Suggestion, 0, len(raw))
	for _, suggestion := range raw {
		command := sanitize(suggestion.Command)
		if command == "" {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			Command:     command,
			Explanation: sanitizeExplanation(suggestion.Explanation),
		})
	}

	return suggestions, nil
}

func stripFences(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}

func decodeSuggestions(content string) ([]Suggestion, error) {
	candidates := []string{content}
	if start, end := strings.Index(content, "["), strings.LastIndex(content, "]"); start >= 0 && end > start {
		candidates = append(candidates, content[start:end+1])
	}

	var firstErr error
	for _, candidate := range candidates {
		suggestions, err := unmarshalSuggestions([]byte(candidate))
		if err == nil {
			return suggestions, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}

	return nil, firstErr
}

func unmarshalSuggestions(data []byte) ([]Suggestion, error) {
	var suggestions []Suggestion
	if err := json.Unmarshal(data, &suggestions); err == nil {
		return suggestions, nil
	}

	var wrapper struct {
		Commands []Suggestion `json:"commands"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.Commands != nil {
		return wrapper.Commands, nil
	}

	return nil, errors.New("response is not a JSON array of commands")
}

func sanitize(command string) string {
	command = ansi.Strip(command)
	command = strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, command)
	command = strings.TrimSpace(command)

	if utf8.RuneCountInString(command) > maxCommandRunes {
		command = string([]rune(command)[:maxCommandRunes])
	}

	return command
}

// Explanations are rendered on one dimmed line next to the command, so
// anything that could move the cursor or wrap the row is removed here.
func sanitizeExplanation(explanation string) string {
	explanation = ansi.Strip(explanation)
	explanation = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return -1
		}
		return r
	}, explanation)
	explanation = strings.Join(strings.Fields(explanation), " ")

	if utf8.RuneCountInString(explanation) > maxExplanationRunes {
		explanation = strings.TrimSpace(string([]rune(explanation)[:maxExplanationRunes]))
	}

	return explanation
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
