package cli

import (
	"io"
	"strings"
	"testing"
)

func TestConfirmLoop(t *testing.T) {
	tests := []struct {
		name, keys, want string
		send             bool
	}{
		{"enter sends", "\r", "fix it", true},
		{"other keys are ignored", "xy\r", "fix it", true},
		{"esc cancels", "\x1b", "", false},
		{"ctrl-c cancels", "\x03", "", false},
		{"closed input cancels", "", "", false},
		{"edit appends", "e now\r", "fix it now", true},
		{"edit backspace", "e\x7f\x7fmore\r", "fix more", true},
		{"edit ctrl-w", "e\x17that\r", "fix that", true},
		{"edit ctrl-u then type", "e\x15hello\r", "hello", true},
		{"edit to empty cancels", "e\x15\r", "", false},
		{"edit esc cancels", "e!\x1b", "", false},
		{"unicode backspace", "e ă\x7fa\r", "fix it a", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, send := confirmLoop(strings.NewReader(tt.keys), io.Discard, "fix it")
			if got != tt.want || send != tt.send {
				t.Errorf("got (%q, %v), want (%q, %v)", got, send, tt.want, tt.send)
			}
		})
	}
}
