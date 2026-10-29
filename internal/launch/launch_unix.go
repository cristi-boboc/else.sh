//go:build !windows

// Package launch starts an agent in the foreground of the current terminal.
package launch

import (
	"os"
	"syscall"
)

// Run replaces the current process with the agent so it owns the terminal
// and receives signals directly. It only returns on failure.
func Run(path string, args []string) (int, error) {
	return 1, syscall.Exec(path, append([]string{path}, args...), os.Environ())
}
