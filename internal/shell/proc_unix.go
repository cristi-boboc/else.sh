//go:build !windows

package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// parent returns the parent pid and executable name of pid.
func parent(pid int) (int, string, error) {
	// Linux: /proc avoids depending on ps.
	if stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
		s := string(stat)
		open, close := strings.IndexByte(s, '('), strings.LastIndexByte(s, ')')
		if open >= 0 && close > open {
			fields := strings.Fields(s[close+1:])
			if len(fields) >= 2 {
				ppid, err := strconv.Atoi(fields[1])
				return ppid, s[open+1 : close], err
			}
		}
	}
	out, err := exec.Command("ps", "-o", "ppid=", "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, "", err
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return 0, "", fmt.Errorf("unexpected ps output %q", out)
	}
	ppid, err := strconv.Atoi(fields[0])
	return ppid, strings.Join(fields[1:], " "), err
}
