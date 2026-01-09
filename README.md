# shelp

Your AI-powered shell assistant. Convert natural language to safe, executable shell commands.

## Features

- **Natural Language Input**: Describe what you want in plain English
- **Safety First**: Dangerous commands are blocked automatically
- **User Confirmation**: Always asks before executing any command
- **BYOK**: Bring Your Own Key - use any OpenAI-compatible API
- **Shell Detection**: Generates commands compatible with your shell (bash, zsh, fish)

## Installation

### Homebrew (Recommended)

```bash
brew tap xqsit94/shelp
brew install shelp
```

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/xqsit94/shelp/main/install.sh | bash
```

### Go Install

```bash
go install github.com/xqsit94/shelp@latest
```

### From Source

```bash
git clone https://github.com/xqsit94/shelp.git
cd shelp
go build -o shelp
mv shelp ~/.local/bin/
```

## Usage

### First Time Setup

On first run, shelp will ask for your AI provider configuration:

```bash
shelp "list all files"

# 🚀 Welcome to shelp! Let's set up your AI provider.
#
# Enter AI API URL (e.g., https://openrouter.ai/api/v1/chat/completions):
# Enter API Key: ****
# Enter model name (e.g., anthropic/claude-3.5-sonnet):
```

### Basic Usage

```bash
# Find files
shelp "find all pdf files larger than 10MB"

# System info
shelp "show disk usage for current directory"

# Docker operations
shelp "list all running docker containers"

# Git operations
shelp "show last 5 commits with file changes"
```

### Configuration

```bash
# Update AI URL
shelp config set url https://openrouter.ai/api/v1/chat/completions

# Update API key (secure input)
shelp config set key

# Update model
shelp config set model anthropic/claude-3.5-sonnet

# Show current configuration
shelp config show

# Reset all configuration
shelp config reset
```

## Supported AI Providers

shelp works with any OpenAI-compatible chat completions API:

- [OpenRouter](https://openrouter.ai) - Access multiple AI models
- [OpenAI](https://openai.com) - GPT-4, GPT-3.5
- [Anthropic](https://anthropic.com) - Claude (via proxy)
- [Together AI](https://together.ai) - Various open models
- [Groq](https://groq.com) - Fast inference
- Local models via [Ollama](https://ollama.ai), [LM Studio](https://lmstudio.ai), etc.

## Safety Features

shelp includes multiple safety mechanisms:

### Blocked Commands

The following patterns are automatically blocked:

- `rm -rf /` or `rm -rf ~/` - Recursive deletion of system/home
- Fork bombs
- Commands with `--no-preserve-root`
- `dd` to system drives
- `chmod 777 /`
- Piping curl/wget directly to shell

### Risk Levels

Commands are categorized by risk:

- **Safe** (green): Read-only operations, simple commands
- **Caution** (yellow): sudo, rm, chmod, system services
- **Danger** (red): Blocked commands

## Examples

### Finding Files

```bash
$ shelp "find all javascript files modified in the last week"

🔍 Generating commands for: find all javascript files modified in the last week
🐚 Detected shell: zsh

Generated command:
┌────────────────────────────────────────────────────────────┐
│ find . -name "*.js" -mtime -7                              │
└────────────────────────────────────────────────────────────┘
Risk: ✅ safe

Execute this command? (y/n):
```

### Multi-Command Operations

```bash
$ shelp "create a project directory and initialize git"

Command 1 of 2:
┌────────────────────────────────────────────────────────────┐
│ mkdir -p myproject                                         │
└────────────────────────────────────────────────────────────┘
Risk: ✅ safe

Execute this command? (y/n): y
✅ Command completed successfully

Command 2 of 2:
┌────────────────────────────────────────────────────────────┐
│ cd myproject && git init                                   │
└────────────────────────────────────────────────────────────┘
Risk: ✅ safe

Execute this command? (y/n):
```

## Configuration File

Configuration is stored in `~/.shelp/config.json`:

```json
{
  "ai_url": "https://openrouter.ai/api/v1/chat/completions",
  "api_key": "sk-or-...",
  "model": "anthropic/claude-3.5-sonnet"
}
```

File permissions are set to `0600` (owner read/write only) for security.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) for details.
