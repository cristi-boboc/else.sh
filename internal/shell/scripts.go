package shell

import (
	"fmt"
	"strings"

	"github.com/cristi-boboc/else.sh/internal/prompt"
)

// The hooks hand unknown commands to `else.sh handle`. Exit status 127 means
// "not a prompt": the hook then falls back to the previous handler (e.g. the
// distro's command-not-found package) or the shell's normal error. Each hook
// only acts in interactive shells, so scripts with missing commands behave
// exactly as before.

const bashHook = `# else.sh hook for bash (>= 4.0) — https://github.com/cristi-boboc/else.sh
if [[ -z "${__ELSE_SH_HOOKED:-}" ]]; then
  if declare -F command_not_found_handle >/dev/null; then
    eval "$(declare -f command_not_found_handle | sed '1s/command_not_found_handle/__else_sh_prev_handle/')"
  fi
  __ELSE_SH_HOOKED=1
fi
command_not_found_handle() {
  if [[ $- == *i* && -z "${ELSE_SH_DISABLE:-}" ]]; then
    __EXE__ handle -- "$@"
    local rc=$?
    [[ $rc -ne 127 ]] && return $rc
  fi
  if declare -F __else_sh_prev_handle >/dev/null; then
    __else_sh_prev_handle "$@"
    return $?
  fi
  printf 'bash: %s: command not found\n' "$1" >&2
  return 127
}
# Lines that start with a question word can name a real command, so they are
# rewritten on Enter, before the shell runs them.
__ELSE_SH_SW=" __SW__ " __ELSE_SH_QW=" __QW__ "
__else_sh_rewrite() {
  [[ -n "${ELSE_SH_DISABLE:-}" ]] && return
  local line="${READLINE_LINE#"${READLINE_LINE%%[![:space:]]*}"}"
  [[ -z "$line" || "$line" == *[\|\;\&\<\>\__BT__\$]* ]] && return
  local first="${line%% *}"; first="${first%%\'*}"; first="${first,,}"
  local -a words; read -ra words <<< "$line"
  if { [[ "$__ELSE_SH_SW" == *" $first "* ]] && (( ${#words[@]} >= 3 )); } ||
     { [[ "$line" == *\? && "$__ELSE_SH_QW" == *" $first "* ]]; }; then
    READLINE_LINE="__ASK__ ask -- '${line//\'/\'\\\'\'}'"
    READLINE_POINT=${#READLINE_LINE}
  fi
}
if [[ $- == *i* ]] && (( BASH_VERSINFO[0] >= 4 )); then
  bind -x '"\C-x\C-]": __else_sh_rewrite'
  bind '"\C-m": "\C-x\C-]\C-j"'
  bind -m vi-insert -x '"\C-x\C-]": __else_sh_rewrite' 2>/dev/null
  bind -m vi-insert '"\C-m": "\C-x\C-]\C-j"' 2>/dev/null
fi
`

const zshHook = `# else.sh hook for zsh — https://github.com/cristi-boboc/else.sh
if [[ -z "${__ELSE_SH_HOOKED:-}" ]]; then
  if (( $+functions[command_not_found_handler] )); then
    functions[__else_sh_prev_handler]=$functions[command_not_found_handler]
  fi
  __ELSE_SH_HOOKED=1
fi
__NOMATCH__command_not_found_handler() {
  if [[ -o interactive && -z "${ELSE_SH_DISABLE:-}" ]]; then
    __EXE__ handle -- "$@"
    local rc=$?
    (( rc != 127 )) && return $rc
  fi
  if (( $+functions[__else_sh_prev_handler] )); then
    __else_sh_prev_handler "$@"
    return $?
  fi
  print -u2 -r -- "zsh: command not found: $1"
  return 127
}
# Lines that start with a question word can name a real command (macOS has
# /usr/bin/what), so they are rewritten on Enter, before the shell runs them.
typeset -ga __else_sh_sw=(__SW__) __else_sh_qw=(__QW__)
__else_sh_accept_line() {
  if [[ -z "${ELSE_SH_DISABLE:-}" ]]; then
    local line="${BUFFER##[[:space:]]#}"
    local first="${${(L)line%% *}%%\'*}"
    local -a words=(${=line})
    if [[ -n "$line" && "$line" != *[\|\;\&\<\>\__BT__\$]* ]]; then
      local lead=0
      (( ${__else_sh_sw[(Ie)$first]} && ${#words} >= 3 )) && lead=1
      [[ "$line" == *\? ]] && (( ${__else_sh_qw[(Ie)$first]} )) && lead=1
      (( lead )) && BUFFER="__ASK__ ask -- ${(qq)line}"
    fi
  fi
  if (( ${+widgets[__else_sh_orig_accept_line]} )); then
    zle __else_sh_orig_accept_line "$@"
  else
    zle .accept-line "$@"
  fi
}
if [[ "${widgets[accept-line]}" == user:* && "${widgets[accept-line]}" != user:__else_sh_accept_line ]]; then
  zle -A accept-line __else_sh_orig_accept_line
fi
zle -N accept-line __else_sh_accept_line
`

const fishHook = `# else.sh hook for fish — https://github.com/cristi-boboc/else.sh
if not set -q __ELSE_SH_HOOKED
    if functions -q fish_command_not_found
        functions -c fish_command_not_found __else_sh_prev_handler
    end
    set -g __ELSE_SH_HOOKED 1
end
function fish_command_not_found
    if status is-interactive; and not set -q ELSE_SH_DISABLE
        __EXE__ handle -- $argv
        set -l rc $status
        if test $rc -ne 127
            return $rc
        end
    end
    if functions -q __else_sh_prev_handler
        __else_sh_prev_handler $argv
    else
        __fish_default_command_not_found_handler $argv
    end
end
# Lines that start with a question word can name a real command, so they are
# rewritten on Enter, before the shell runs them.
function __else_sh_enter
    set -l line (string trim -- (commandline | string collect))
    if not set -q ELSE_SH_DISABLE; and test -n "$line"; and not string match -qr '[|;&<>__BT__$]' -- $line
        set -l first (string split -m1 ' ' -- $line)[1]
        set first (string lower -- (string replace -r "'.*" '' -- $first))
        set -l n (count (string split -n ' ' -- $line))
        if begin; contains -- $first __SW__; and test $n -ge 3; end
            or begin; string match -qr '\?$' -- $line; and contains -- $first __QW__; end
            commandline -r -- "__ASK__ ask -- "(string escape -- $line)
        end
    end
    commandline -f execute
end
bind \r __else_sh_enter
bind -M insert \r __else_sh_enter 2>/dev/null
`

const powershellHook = `# else.sh hook for PowerShell — https://github.com/cristi-boboc/else.sh
$global:__ElseShExe = __EXE__
$ExecutionContext.InvokeCommand.CommandNotFoundAction = {
    param([string]$Name, [System.Management.Automation.CommandLookupEventArgs]$EventArgs)
    if ($EventArgs.CommandOrigin -ne 'Runspace' -or $env:ELSE_SH_DISABLE) { return }
    if ($Name -match '^(get-|\.|\\|/)' -or $Name -match '[\\/]') { return }
    $cmdName = $Name
    $EventArgs.StopSearch = $true
    $EventArgs.CommandScriptBlock = {
        & $global:__ElseShExe handle $cmdName @args
        if ($LASTEXITCODE -eq 127) {
            Write-Error -Category ObjectNotFound -Message "The term '$cmdName' is not recognized as a name of a cmdlet, function, script file, or executable program."
        }
    }.GetNewClosure()
}
# Lines that start with a question word can name a real command, so they are
# rewritten on Enter, before PowerShell runs them.
if (Get-Module PSReadLine) {
    Set-PSReadLineKeyHandler -Key Enter -BriefDescription ElseShAcceptLine -ScriptBlock {
        $line = $null; $cursor = $null
        [Microsoft.PowerShell.PSConsoleReadLine]::GetBufferState([ref]$line, [ref]$cursor)
        $t = $line.Trim()
        if (-not $env:ELSE_SH_DISABLE -and $t -and $t -notmatch '[|;&<>__BT__$]') {
            $words = @($t -split '\s+')
            $first = ($words[0] -replace "'.*", '').ToLower()
            if ((@(__SW__) -contains $first -and $words.Count -ge 3) -or ($t.EndsWith('?') -and @(__QW__) -contains $first)) {
                [Microsoft.PowerShell.PSConsoleReadLine]::Replace(0, $line.Length, "& __ASK__ ask -- '" + ($t -replace "'", "''") + "'")
            }
        }
        [Microsoft.PowerShell.PSConsoleReadLine]::AcceptLine()
    }
}
`

// Init renders the hook script for a shell. exe is the absolute path of the
// else.sh binary and ask the command written into rewritten lines ("else.sh"
// when that resolves to exe, so history stays readable); noNomatch applies
// only to zsh.
func Init(name, exe, ask string, noNomatch bool) (string, error) {
	var tpl string
	var quote func(string) string
	list := func(words []string) string { return strings.Join(words, " ") }
	switch name {
	case Bash:
		tpl, quote = bashHook, posixQuote
	case Zsh:
		tpl, quote = zshHook, posixQuote
	case Fish:
		tpl, quote = fishHook, fishQuote
	case PowerShell:
		tpl, quote = powershellHook, psQuote
		list = func(words []string) string {
			q := make([]string, len(words))
			for i, w := range words {
				q[i] = psQuote(w)
			}
			return strings.Join(q, ",")
		}
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", name, strings.Join(Names, ", "))
	}
	if ask != "else.sh" {
		ask = quote(ask)
	}
	nomatch := ""
	if name == Zsh && noNomatch {
		// Unmatched globs such as "why?" would otherwise abort the line before
		// the handler runs.
		nomatch = "setopt no_nomatch\n"
	}
	return strings.NewReplacer(
		"__BT__", "`",
		"__NOMATCH__", nomatch,
		"__EXE__", quote(exe),
		"__ASK__", ask,
		"__SW__", list(prompt.LeadWords),
		"__QW__", list(prompt.QuestionWords),
	).Replace(tpl), nil
}

func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func fishQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return "'" + strings.ReplaceAll(s, "'", `\'`) + "'"
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
