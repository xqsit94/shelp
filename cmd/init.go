package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var initShells = []string{"zsh", "bash", "fish", "powershell"}

func InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <" + strings.Join(initShells, "|") + ">",
		Short: "Print the shell integration snippet",
		Long: `Print the shell key binding snippet for a shell.

The snippet binds ctrl+g to a widget that sends the current command line to
shelp as the query and replaces it with the generated commands, so a line can
be written in English and turned into shell commands in place.

Load it from your shell startup file:

  zsh         eval "$(shelp init zsh)"                                   ~/.zshrc
  bash        eval "$(shelp init bash)"                                  ~/.bashrc
  fish        shelp init fish | source                                   ~/.config/fish/config.fish
  powershell  Invoke-Expression (& shelp init powershell | Out-String)   $PROFILE`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			snippet, ok := initSnippets[strings.ToLower(args[0])]
			if !ok {
				return &ExitError{Code: 1, Err: fmt.Errorf("unsupported shell %q, supported shells are: %s", args[0], strings.Join(initShells, ", "))}
			}

			fmt.Fprint(cmd.OutOrStdout(), snippet)

			return nil
		},
	}
}

var initSnippets = map[string]string{
	"zsh":        zshSnippet,
	"bash":       bashSnippet,
	"fish":       fishSnippet,
	"powershell": powershellSnippet,
}

// The query is passed after -- so that a line starting with a dash is taken as
// the query instead of as flags, and stderr is sent to the terminal so that
// failures stay visible while stdout is being captured.
const zshSnippet = `# shelp shell integration for zsh: ctrl+g rewrites the current line.
_shelp_widget() {
  emulate -L zsh

  local query=$BUFFER
  if [[ -z ${query//[[:space:]]/} ]]; then
    return 0
  fi

  zle -M "shelp: generating…"

  local output
  output=$(command shelp -p -- "$query" 2>/dev/tty)
  local code=$?

  zle -M ""

  if (( code != 0 )) || [[ -z $output ]]; then
    return 0
  fi

  local -a commands
  commands=(${(f)output})
  BUFFER=${(j: && :)commands}
  CURSOR=${#BUFFER}
}

zle -N _shelp_widget
bindkey '^G' _shelp_widget

# Rebind: copy this snippet into ~/.zshrc and change '^G' to another key.
`

const bashSnippet = `# shelp shell integration for bash: ctrl+g rewrites the current line.
_shelp_widget() {
  local query=$READLINE_LINE
  if [[ -z ${query//[[:space:]]/} ]]; then
    return 0
  fi

  printf 'shelp: generating…' >/dev/tty

  local output
  output=$(command shelp -p -- "$query" 2>/dev/tty)
  local code=$?

  printf '\r\033[2K' >/dev/tty

  if [[ $code -ne 0 || -z $output ]]; then
    return 0
  fi

  local joined= line
  while IFS= read -r line; do
    if [[ -z $line ]]; then
      continue
    fi
    if [[ -n $joined ]]; then
      joined+=" && $line"
    else
      joined=$line
    fi
  done <<<"$output"

  READLINE_LINE=$joined
  READLINE_POINT=${#READLINE_LINE}
}

bind -x '"\C-g": _shelp_widget'

# Rebind: copy this snippet into ~/.bashrc and change "\C-g" to another key.
`

const fishSnippet = `# shelp shell integration for fish: ctrl+g rewrites the current line.
function _shelp_widget --description "Rewrite the command line with shelp"
    set -l query (commandline)
    set -l trimmed (string trim -- "$query")
    if test -z "$trimmed"
        return 0
    end

    printf 'shelp: generating…' >/dev/tty

    set -l output (command shelp -p -- "$query" 2>/dev/tty)
    set -l code $status

    printf '\r\033[2K' >/dev/tty

    if test $code -ne 0; or test (count $output) -eq 0
        return 0
    end

    commandline -r -- (string join ' && ' -- $output)
    commandline -f repaint
end

bind \cg _shelp_widget

# Rebind: copy this snippet into ~/.config/fish/config.fish and change \cg.
`

const powershellSnippet = `# shelp shell integration for PowerShell: ctrl+g rewrites the current line.
Set-PSReadLineKeyHandler -Chord 'Ctrl+g' -Description 'Rewrite the command line with shelp' -ScriptBlock {
    $line = ''
    $cursor = 0
    [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
    if ([string]::IsNullOrWhiteSpace($line)) {
        return
    }

    $commands = @(shelp -p -- $line)
    if ($LASTEXITCODE -ne 0 -or $commands.Count -eq 0) {
        return
    }

    [Microsoft.PowerShell.PSConsoleReadLine]::Replace(0, $line.Length, ($commands -join '; '))
}

# Rebind: copy this snippet into $PROFILE and change 'Ctrl+g' to another chord.
`
