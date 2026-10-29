// Package shell detects the user's shell, renders its hook and installs it in the rc file.
package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Supported shell names.
const (
	Bash       = "bash"
	Zsh        = "zsh"
	Fish       = "fish"
	PowerShell = "powershell"
)

// Names lists the supported shells.
var Names = []string{Bash, Zsh, Fish, PowerShell}

// Shell is a detected shell.
type Shell struct {
	Name string
	// Exe is the shell executable when known (pwsh vs. powershell.exe matters on Windows).
	Exe string
}

// Normalize maps a process or user-supplied name to a supported shell name.
func Normalize(name string) (string, bool) {
	n := strings.ToLower(filepath.Base(strings.TrimPrefix(name, "-")))
	n = strings.TrimSuffix(n, ".exe")
	switch n {
	case "bash", "zsh", "fish":
		return n, true
	case "pwsh", "powershell", "pwsh-preview":
		return PowerShell, true
	}
	return "", false
}

// Detect finds the shell this process was started from: ELSE_SH_SHELL, then
// the nearest shell among parent processes, then $SHELL.
func Detect() (Shell, error) {
	if s := os.Getenv("ELSE_SH_SHELL"); s != "" {
		if n, ok := Normalize(s); ok {
			return Shell{Name: n, Exe: s}, nil
		}
		return Shell{}, fmt.Errorf("ELSE_SH_SHELL=%q is not a supported shell", s)
	}
	pid := os.Getppid()
	for range 8 {
		if pid <= 1 {
			break
		}
		ppid, exe, err := parent(pid)
		if err != nil {
			break
		}
		if n, ok := Normalize(exe); ok {
			return Shell{Name: n, Exe: exe}, nil
		}
		pid = ppid
	}
	if s := os.Getenv("SHELL"); s != "" {
		if n, ok := Normalize(s); ok {
			return Shell{Name: n, Exe: s}, nil
		}
	}
	if runtime.GOOS == "windows" {
		return Shell{Name: PowerShell, Exe: "powershell.exe"}, nil
	}
	return Shell{}, fmt.Errorf("could not detect a supported shell (%s)", strings.Join(Names, ", "))
}
