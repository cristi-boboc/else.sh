// Package agent knows which coding agents exist and how to start them with a prompt.
package agent

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/cristi-boboc/else.sh/internal/config"
)

// Placeholder is replaced with the user's prompt in an agent's arguments.
const Placeholder = "{prompt}"

// Agent describes how to launch one agent CLI interactively with a first prompt.
type Agent struct {
	Name    string
	Command string
	Args    []string
	Builtin bool
}

// Builtins are listed in default-preference order: the first one installed
// becomes the default agent.
var Builtins = []Agent{
	{Name: "claude", Command: "claude", Args: []string{Placeholder}},
	{Name: "codex", Command: "codex", Args: []string{Placeholder}},
	{Name: "gemini", Command: "gemini", Args: []string{"-i", Placeholder}},
	{Name: "copilot", Command: "copilot", Args: []string{"-i", Placeholder}},
	{Name: "cursor", Command: "cursor-agent", Args: []string{Placeholder}},
	{Name: "opencode", Command: "opencode", Args: []string{"--prompt", Placeholder}},
	{Name: "qwen", Command: "qwen", Args: []string{"-i", Placeholder}},
	{Name: "goose", Command: "goose", Args: []string{"run", "--interactive", "-t", Placeholder}},
	{Name: "aider", Command: "aider", Args: []string{"--message", Placeholder}},
}

// All returns the built-in agents with config overrides applied, followed by
// custom agents in name order.
func All(cfg config.Config) []Agent {
	out := make([]Agent, 0, len(Builtins)+len(cfg.Agents))
	for _, b := range Builtins {
		if o, ok := cfg.Agents[b.Name]; ok {
			b = merge(b, o)
		}
		b.Args = slices.Clone(b.Args)
		out = append(out, b)
	}
	var custom []string
	for name := range cfg.Agents {
		if !slices.ContainsFunc(Builtins, func(b Agent) bool { return b.Name == name }) {
			custom = append(custom, name)
		}
	}
	slices.Sort(custom)
	for _, name := range custom {
		out = append(out, merge(Agent{Name: name, Command: name, Args: []string{Placeholder}}, cfg.Agents[name]))
	}
	return out
}

func merge(a Agent, o config.Agent) Agent {
	if o.Command != "" {
		a.Command = o.Command
	}
	if o.Args != nil {
		a.Args = slices.Clone(o.Args)
	}
	return a
}

// Find returns the agent with the given name.
func Find(all []Agent, name string) (Agent, bool) {
	i := slices.IndexFunc(all, func(a Agent) bool { return a.Name == name })
	if i < 0 {
		return Agent{}, false
	}
	return all[i], true
}

// LookPath resolves the agent's executable on PATH.
func (a Agent) LookPath() (string, bool) {
	p, err := exec.LookPath(a.Command)
	return p, err == nil
}

// Detect returns the agents that are installed, in preference order.
func Detect(all []Agent) []Agent {
	var found []Agent
	for _, a := range all {
		if _, ok := a.LookPath(); ok {
			found = append(found, a)
		}
	}
	return found
}

// Argv builds the agent's arguments for a prompt. If no argument contains the
// placeholder, the prompt is appended.
func (a Agent) Argv(prompt string) []string {
	args := make([]string, 0, len(a.Args)+1)
	used := false
	for _, arg := range a.Args {
		if strings.Contains(arg, Placeholder) {
			used = true
			arg = strings.ReplaceAll(arg, Placeholder, prompt)
		}
		args = append(args, arg)
	}
	if !used {
		args = append(args, prompt)
	}
	return args
}

// Resolve picks the agent to run: the configured default if installed,
// otherwise the first installed agent.
func Resolve(cfg config.Config) (Agent, string, error) {
	all := All(cfg)
	if cfg.DefaultAgent != "" {
		if a, ok := Find(all, cfg.DefaultAgent); ok {
			if p, ok := a.LookPath(); ok {
				return a, p, nil
			}
		}
	}
	for _, a := range all {
		if p, ok := a.LookPath(); ok {
			return a, p, nil
		}
	}
	return Agent{}, "", fmt.Errorf("no agent found on PATH (looked for %s)", names(all))
}

func names(all []Agent) string {
	n := make([]string, len(all))
	for i, a := range all {
		n[i] = a.Command
	}
	return strings.Join(n, ", ")
}
