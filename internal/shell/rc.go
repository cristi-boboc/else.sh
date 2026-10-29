package shell

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	beginMarker = "# >>> else.sh >>>"
	endMarker   = "# <<< else.sh <<<"
)

// RCFile returns the startup file the hook is installed into.
func RCFile(s Shell) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch s.Name {
	case Bash:
		if runtime.GOOS == "darwin" {
			// Terminal.app starts login shells, which read .bash_profile.
			return filepath.Join(home, ".bash_profile"), nil
		}
		return filepath.Join(home, ".bashrc"), nil
	case Zsh:
		dir := os.Getenv("ZDOTDIR")
		if dir == "" {
			dir = home
		}
		return filepath.Join(dir, ".zshrc"), nil
	case Fish:
		dir := os.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			dir = filepath.Join(home, ".config")
		}
		return filepath.Join(dir, "fish", "conf.d", "else.sh.fish"), nil
	case PowerShell:
		return powershellProfile(s, home)
	}
	return "", fmt.Errorf("unsupported shell %q", s.Name)
}

func powershellProfile(s Shell, home string) (string, error) {
	exe := s.Exe
	if exe == "" || !filepath.IsAbs(exe) {
		for _, c := range []string{exe, "pwsh", "powershell"} {
			if c == "" {
				continue
			}
			if p, err := exec.LookPath(c); err == nil {
				exe = p
				break
			}
		}
	}
	if exe != "" {
		out, err := exec.Command(exe, "-NoLogo", "-NoProfile", "-NonInteractive", "-Command", "$PROFILE.CurrentUserAllHosts").Output()
		if p := strings.TrimSpace(string(out)); err == nil && p != "" {
			return p, nil
		}
	}
	if runtime.GOOS == "windows" {
		dir := "PowerShell"
		if strings.Contains(strings.ToLower(filepath.Base(exe)), "powershell") {
			dir = "WindowsPowerShell"
		}
		return filepath.Join(home, "Documents", dir, "profile.ps1"), nil
	}
	return filepath.Join(home, ".config", "powershell", "profile.ps1"), nil
}

// HookLine is the rc-file snippet that loads the hook at shell startup.
func HookLine(name, exe string) string {
	switch name {
	case Bash, Zsh:
		q := posixQuote(exe)
		return fmt.Sprintf("[ -x %s ] && eval \"$(%s init %s)\"", q, q, name)
	case Fish:
		q := fishQuote(exe)
		return fmt.Sprintf("test -x %s; and %s init fish | source", q, q)
	case PowerShell:
		q := psQuote(exe)
		return fmt.Sprintf("if (Test-Path %s) { & %s init powershell | Out-String | Invoke-Expression }", q, q)
	}
	return ""
}

// Install writes the hook block into the shell's rc file, replacing an
// existing block. It reports the file and whether it changed.
func Install(s Shell, exe string) (string, bool, error) {
	path, err := RCFile(s)
	if err != nil {
		return "", false, err
	}
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return path, false, err
	}
	block := beginMarker + "\n" + HookLine(s.Name, exe) + "\n" + endMarker + "\n"
	updated := replaceBlock(string(old), block)
	if updated == string(old) {
		return path, false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, false, err
	}
	return path, true, os.WriteFile(path, []byte(updated), 0o644)
}

// Uninstall removes the hook block (or, for fish, the whole conf.d file).
func Uninstall(s Shell) (string, bool, error) {
	path, err := RCFile(s)
	if err != nil {
		return "", false, err
	}
	old, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return path, false, nil
	}
	if err != nil {
		return path, false, err
	}
	updated := replaceBlock(string(old), "")
	if updated == string(old) {
		return path, false, nil
	}
	if s.Name == Fish && strings.TrimSpace(updated) == "" {
		return path, true, os.Remove(path)
	}
	return path, true, os.WriteFile(path, []byte(updated), 0o644)
}

// Installed reports whether the rc file contains the hook block.
func Installed(s Shell) (string, bool) {
	path, err := RCFile(s)
	if err != nil {
		return "", false
	}
	data, err := os.ReadFile(path)
	return path, err == nil && strings.Contains(string(data), beginMarker)
}

// replaceBlock swaps the marked block in content for block (which may be
// empty), appending it when no block exists.
func replaceBlock(content, block string) string {
	start := strings.Index(content, beginMarker)
	if start < 0 {
		if block == "" {
			return content
		}
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content != "" {
			content += "\n"
		}
		return content + block
	}
	end := strings.Index(content[start:], endMarker)
	if end < 0 {
		return content
	}
	end += start + len(endMarker)
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return content[:start] + block + content[end:]
}
