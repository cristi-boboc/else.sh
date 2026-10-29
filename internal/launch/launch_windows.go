//go:build windows

// Package launch starts an agent in the foreground of the current terminal.
package launch

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
)

// Run starts the agent attached to the console and waits for it, returning
// its exit code.
func Run(path string, args []string) (int, error) {
	cmd := exec.Command(path, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	// Ctrl+C goes to the whole console group; let the agent handle it.
	signal.Ignore(os.Interrupt)
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	if err != nil {
		return 1, err
	}
	return 0, nil
}
