package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	keyCtrlC     = 0x03
	keyCtrlU     = 0x15
	keyCtrlW     = 0x17
	keyEsc       = 0x1b
	keyBackspace = 0x7f
	keyCtrlH     = 0x08
)

// confirm shows the prompt and waits for Enter (send), e (edit) or Esc
// (cancel). It returns the text to send and whether to send it.
func confirm(agentName, text string) (string, bool) {
	fd := int(os.Stdin.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		// Can't read single keys; don't block the hand-off.
		return text, true
	}
	defer term.Restore(fd, state)
	fmt.Fprintf(os.Stderr, "%s %s: %s\r\n", dim("else.sh →"), bold(agentName), text)
	return confirmLoop(bufio.NewReader(os.Stdin), os.Stderr, text)
}

// confirmLoop runs the key handling on any reader, so it can be tested.
// Output uses \r\n because the terminal is in raw mode.
func confirmLoop(r io.RuneReader, w io.Writer, text string) (string, bool) {
	hint := dim("  enter send   e edit   esc cancel")
	clear := "\r\x1b[K"
	fmt.Fprint(w, hint)
	for {
		c, _, err := r.ReadRune()
		if err != nil {
			fmt.Fprint(w, clear)
			return "", false
		}
		switch c {
		case '\r', '\n':
			fmt.Fprint(w, clear)
			return text, true
		case 'e', 'E':
			return edit(r, w, text)
		case keyEsc, keyCtrlC, 'n', 'N', 'q', 'Q':
			fmt.Fprint(w, clear+dim("  cancelled")+"\r\n")
			return "", false
		}
	}
}

// edit is a minimal line editor: type to append, Backspace, Ctrl-W deletes a
// word, Ctrl-U clears, Enter sends, Esc cancels.
func edit(r io.RuneReader, w io.Writer, text string) (string, bool) {
	buf := []rune(text)
	redraw := func() { fmt.Fprintf(w, "\r\x1b[K%s %s", dim("  edit›"), string(buf)) }
	redraw()
	for {
		c, _, err := r.ReadRune()
		if err != nil {
			fmt.Fprint(w, "\r\x1b[K")
			return "", false
		}
		switch {
		case c == '\r' || c == '\n':
			fmt.Fprint(w, "\r\x1b[K")
			out := strings.TrimSpace(string(buf))
			return out, out != ""
		case c == keyEsc || c == keyCtrlC:
			fmt.Fprint(w, "\r\x1b[K"+dim("  cancelled")+"\r\n")
			return "", false
		case c == keyBackspace || c == keyCtrlH:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
		case c == keyCtrlU:
			buf = buf[:0]
		case c == keyCtrlW:
			s := strings.TrimRight(string(buf), " ")
			if i := strings.LastIndexByte(s, ' '); i >= 0 {
				buf = []rune(s[:i+1])
			} else {
				buf = buf[:0]
			}
		case c >= ' ':
			buf = append(buf, c)
		}
		redraw()
	}
}
