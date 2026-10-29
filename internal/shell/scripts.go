package shell

import (
	"fmt"
	"strings"
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
`

// Init renders the hook script for a shell. exe is the absolute path of the
// else.sh binary; noNomatch applies only to zsh.
func Init(name, exe string, noNomatch bool) (string, error) {
	var tpl, quoted string
	switch name {
	case Bash:
		tpl, quoted = bashHook, posixQuote(exe)
	case Zsh:
		tpl, quoted = zshHook, posixQuote(exe)
	case Fish:
		tpl, quoted = fishHook, fishQuote(exe)
	case PowerShell:
		tpl, quoted = powershellHook, psQuote(exe)
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: %s)", name, strings.Join(Names, ", "))
	}
	nomatch := ""
	if name == Zsh && noNomatch {
		// Unmatched globs such as "why?" would otherwise abort the line before
		// the handler runs.
		nomatch = "setopt no_nomatch\n"
	}
	tpl = strings.ReplaceAll(tpl, "__NOMATCH__", nomatch)
	return strings.ReplaceAll(tpl, "__EXE__", quoted), nil
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
