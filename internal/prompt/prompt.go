// Package prompt decides whether an unknown command line was meant for an agent.
package prompt

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Options tune the decision.
type Options struct {
	MinWords  int
	TypoGuard bool
	// Commands returns executable names on PATH; nil uses the real PATH.
	Commands func() map[string]struct{}
}

// intentWords always start a prompt, even if they resemble a real command
// (e.g. "why" is one edit away from "who").
var intentWords = map[string]bool{
	"why": true, "how": true, "what": true, "where": true, "when": true, "which": true,
	"who": true, "whats": true, "can": true, "could": true, "would": true, "should": true,
	"please": true, "pls": true, "explain": true, "fix": true, "write": true, "add": true,
	"create": true, "refactor": true, "help": true, "show": true, "tell": true,
	"summarize": true, "summarise": true, "review": true, "generate": true,
	"implement": true, "debug": true, "build": true, "make": true, "run": true,
	"update": true, "remove": true, "delete": true, "rename": true, "migrate": true,
	"optimize": true, "document": true, "describe": true, "list": true, "find": true,
	"is": true, "are": true, "does": true, "do": true, "hey": true, "hi": true,
}

// Decision is the outcome for one command line.
type Decision struct {
	Accept bool
	Reason string
	Prompt string
}

// Decide reports whether words (the command and its arguments) look like a prompt.
func Decide(words []string, o Options) Decision {
	if len(words) == 0 || strings.TrimSpace(words[0]) == "" {
		return Decision{Reason: "empty line"}
	}
	first := words[0]
	text := strings.Join(words, " ")
	if strings.ContainsAny(first, `/\`) || strings.HasPrefix(first, "-") || strings.HasPrefix(first, ".") {
		return Decision{Reason: "first word looks like a path or flag"}
	}
	key := strings.ToLower(strings.TrimRight(first, "?!.,:"))
	if intentWords[key] {
		return Decision{Accept: true, Reason: "starts with an intent word", Prompt: text}
	}
	if len(words) < max(o.MinWords, 1) && !strings.HasSuffix(text, "?") {
		return Decision{Reason: "too few words"}
	}
	if o.TypoGuard {
		cmds := o.Commands
		if cmds == nil {
			cmds = PathCommands
		}
		if near := nearCommand(strings.ToLower(first), cmds()); near != "" {
			return Decision{Reason: "looks like a typo of " + near}
		}
	}
	return Decision{Accept: true, Reason: "looks like natural language", Prompt: text}
}

func nearCommand(word string, cmds map[string]struct{}) string {
	if len(word) < 3 {
		return ""
	}
	for c := range cmds {
		if abs(len(c)-len(word)) <= 1 && osaDistance(word, c) == 1 {
			return c
		}
	}
	return ""
}

// osaDistance is the optimal-string-alignment edit distance (Levenshtein plus
// adjacent transpositions), so "gti" -> "git" is 1.
func osaDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	d := make([][]int, len(ra)+1)
	for i := range d {
		d[i] = make([]int, len(rb)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[len(ra)][len(rb)]
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

var (
	pathOnce sync.Once
	pathCmds map[string]struct{}
)

// PathCommands lists executable names found on PATH (lower-cased, without
// Windows extensions).
func PathCommands() map[string]struct{} {
	pathOnce.Do(func() {
		pathCmds = map[string]struct{}{}
		for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := strings.ToLower(e.Name())
				if runtime.GOOS == "windows" {
					ext := filepath.Ext(name)
					switch ext {
					case ".exe", ".cmd", ".bat", ".ps1", ".com":
						name = strings.TrimSuffix(name, ext)
					default:
						continue
					}
				}
				pathCmds[name] = struct{}{}
			}
		}
	})
	return pathCmds
}
