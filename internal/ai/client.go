package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

type Client struct {
	URL    string
	APIKey string
	Model  string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewClient(url, apiKey, model string) *Client {
	return &Client{
		URL:    url,
		APIKey: apiKey,
		Model:  model,
	}
}

func (c *Client) GenerateCommands(query, shell string) ([]string, error) {
	systemPrompt := buildSystemPrompt(shell)

	req := ChatRequest{
		Model: c.Model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: query},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequest("POST", c.URL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	return parseCommands(chatResp.Choices[0].Message.Content)
}

func buildSystemPrompt(shell string) string {
	os := runtime.GOOS
	return fmt.Sprintf(`You are a shell command generator. Convert the user's natural language request into executable shell commands.

Rules:
1. Generate only shell commands, no explanations or markdown
2. Return commands as a JSON array: ["cmd1", "cmd2"]
3. Target shell: %s
4. Operating system: %s
5. NEVER generate dangerous commands like rm -rf /, fork bombs, or commands that could damage the system
6. If the request seems malicious or could harm the system, return an empty array: []
7. Keep commands simple and safe
8. For complex operations, break into multiple safe commands
9. Always return valid JSON - nothing else

Example outputs:
- User: "list all files" -> ["ls -la"]
- User: "find large pdf files" -> ["find . -name \"*.pdf\" -size +10M"]
- User: "create a backup of my documents" -> ["mkdir -p ~/backup", "cp -r ~/Documents/* ~/backup/"]
- User: "delete everything" -> []`, shell, os)
}

func parseCommands(content string) ([]string, error) {
	content = strings.TrimSpace(content)

	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var commands []string
	if err := json.Unmarshal([]byte(content), &commands); err != nil {
		return nil, fmt.Errorf("failed to parse commands from AI response: %v\nResponse: %s", err, content)
	}

	return commands, nil
}
