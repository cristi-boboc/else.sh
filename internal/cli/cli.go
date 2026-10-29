// Package cli implements the else.sh command line.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/cristi-boboc/else.sh/internal/agent"
	"github.com/cristi-boboc/else.sh/internal/config"
	"github.com/cristi-boboc/else.sh/internal/launch"
	"github.com/cristi-boboc/else.sh/internal/prompt"
	"github.com/cristi-boboc/else.sh/internal/shell"
)

// Version is set at build time.
var Version = "dev"

// declined tells the shell hook to fall back to the normal "command not found".
const declined = 127

const usage = `else.sh — type a prompt in your shell, get your coding agent.

Usage:
  else.sh install [shell]     hook into your shell and pick a default agent
  else.sh uninstall [shell]   remove the hook
  else.sh agents              list known agents and which are installed
  else.sh use <agent>         set the default agent
  else.sh ask <prompt...>     send a prompt to the default agent explicitly
  else.sh on | off            enable or disable the hand-off
  else.sh doctor              show what else.sh detected
  else.sh init <shell>        print the hook script (used by your rc file)
  else.sh version

Shells: bash (>= 4), zsh, fish, powershell (pwsh / Windows PowerShell).
`

// Run executes the CLI and returns the process exit code.
func Run(args []string) int {
	if len(args) == 0 {
		fmt.Print(usage)
		return 0
	}
	cmd, rest := args[0], args[1:]
	var err error
	switch cmd {
	case "handle":
		return handle(rest)
	case "ask":
		return ask(rest)
	case "init":
		err = initCmd(rest)
	case "install":
		err = install(rest)
	case "uninstall":
		err = uninstall(rest)
	case "agents", "detect":
		err = agents()
	case "use":
		err = use(rest)
	case "on", "off":
		err = toggle(cmd == "on")
	case "doctor":
		err = doctor()
	case "version", "--version", "-v":
		fmt.Println("else.sh", Version)
	case "help", "--help", "-h":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "else.sh: unknown command %q\n\n%s", cmd, usage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "else.sh:", err)
		return 1
	}
	return 0
}

// handle is called by the shell hook with the unknown command line.
func handle(args []string) int {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	// Never hijack scripts or pipes; ELSE_SH_ASSUME_TTY exists for tests.
	interactive := term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	if !interactive && os.Getenv("ELSE_SH_ASSUME_TTY") == "" {
		return declined
	}
	cfg, err := loadOrInit()
	if err != nil || !cfg.Enabled {
		return declined
	}
	d := prompt.Decide(args, prompt.Options{MinWords: cfg.MinWords, TypoGuard: cfg.TypoGuard})
	if os.Getenv("ELSE_SH_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "else.sh: accept=%v (%s)\n", d.Accept, d.Reason)
	}
	if !d.Accept {
		return declined
	}
	return start(cfg, d.Prompt)
}

func ask(args []string) int {
	text := strings.TrimSpace(strings.Join(args, " "))
	if text == "" {
		fmt.Fprintln(os.Stderr, "usage: else.sh ask <prompt...>")
		return 2
	}
	cfg, err := loadOrInit()
	if err != nil {
		fmt.Fprintln(os.Stderr, "else.sh:", err)
		return 1
	}
	return start(cfg, text)
}

func start(cfg config.Config, text string) int {
	a, path, err := agent.Resolve(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "else.sh:", err)
		return declined
	}
	if cfg.Announce {
		fmt.Fprintf(os.Stderr, "%s %s\n", dim("else.sh →"), bold(a.Name))
	}
	code, err := launch.Run(path, a.Argv(text))
	if err != nil {
		fmt.Fprintf(os.Stderr, "else.sh: starting %s: %v\n", a.Name, err)
	}
	return code
}

// loadOrInit loads the config; on first run it detects agents and saves a
// config with the best one as default.
func loadOrInit() (config.Config, error) {
	cfg, exists, err := config.Load()
	if err != nil {
		return cfg, err
	}
	if exists && cfg.DefaultAgent != "" {
		return cfg, nil
	}
	if found := agent.Detect(agent.All(cfg)); len(found) > 0 {
		cfg.DefaultAgent = found[0].Name
		_, err = config.Save(cfg)
	}
	return cfg, err
}

func initCmd(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: else.sh init <%s>", strings.Join(shell.Names, "|"))
	}
	name, ok := shell.Normalize(args[0])
	if !ok {
		return fmt.Errorf("unsupported shell %q (supported: %s)", args[0], strings.Join(shell.Names, ", "))
	}
	exe, err := self()
	if err != nil {
		return err
	}
	cfg, _, _ := config.Load()
	script, err := shell.Init(name, exe, cfg.ZshNoNomatch)
	if err != nil {
		return err
	}
	fmt.Print(script)
	return nil
}

func pickShell(args []string) (shell.Shell, error) {
	if len(args) > 0 {
		name, ok := shell.Normalize(args[0])
		if !ok {
			return shell.Shell{}, fmt.Errorf("unsupported shell %q (supported: %s)", args[0], strings.Join(shell.Names, ", "))
		}
		return shell.Shell{Name: name, Exe: args[0]}, nil
	}
	return shell.Detect()
}

func install(args []string) error {
	sh, err := pickShell(args)
	if err != nil {
		return err
	}
	exe, err := self()
	if err != nil {
		return err
	}
	rc, changed, err := shell.Install(sh, exe)
	if err != nil {
		return fmt.Errorf("writing %s: %w", rc, err)
	}
	if changed {
		fmt.Printf("%s hooked %s in %s\n", ok(), bold(sh.Name), rc)
	} else {
		fmt.Printf("%s %s already hooked in %s\n", ok(), bold(sh.Name), rc)
	}

	cfg, _, err := config.Load()
	if err != nil {
		return err
	}
	all := agent.All(cfg)
	found := agent.Detect(all)
	if len(found) == 0 {
		fmt.Printf("%s no agent found on PATH. Install one of: %s\n", warn(), agentNames(all))
	} else {
		if _, isFound := agent.Find(found, cfg.DefaultAgent); !isFound {
			cfg.DefaultAgent = found[0].Name
		}
		fmt.Printf("%s agents found: %s — default: %s\n", ok(), agentNames(found), bold(cfg.DefaultAgent))
	}
	path, err := config.Save(cfg)
	if err != nil {
		return err
	}
	fmt.Printf("%s config: %s\n", ok(), path)
	if sh.Name == shell.Bash {
		fmt.Println(dim("  note: needs bash 4 or newer (macOS ships 3.2; `brew install bash`)."))
	}
	fmt.Printf("\nOpen a new terminal (or reload %s), then just type:\n\n  %s\n\n", rc, bold("explain what this repo does"))
	return nil
}

func uninstall(args []string) error {
	sh, err := pickShell(args)
	if err != nil {
		return err
	}
	rc, changed, err := shell.Uninstall(sh)
	if err != nil {
		return err
	}
	if changed {
		fmt.Printf("%s removed the %s hook from %s\n", ok(), sh.Name, rc)
	} else {
		fmt.Printf("%s no hook found in %s\n", ok(), rc)
	}
	return nil
}

func agents() error {
	cfg, err := loadOrInit()
	if err != nil {
		return err
	}
	def, _, _ := agent.Resolve(cfg)
	for _, a := range agent.All(cfg) {
		mark, where := "  ", dim("not installed")
		if p, found := a.LookPath(); found {
			where = p
			if a.Name == def.Name {
				mark = green("* ")
			}
		}
		fmt.Printf("%s%-10s %s\n", mark, a.Name, where)
	}
	return nil
}

func use(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: else.sh use <agent>")
	}
	cfg, _, err := config.Load()
	if err != nil {
		return err
	}
	a, found := agent.Find(agent.All(cfg), args[0])
	if !found {
		return fmt.Errorf("unknown agent %q (see `else.sh agents`, or add it under [agents.%s] in the config)", args[0], args[0])
	}
	if _, onPath := a.LookPath(); !onPath {
		fmt.Printf("%s %s is not on PATH yet\n", warn(), a.Command)
	}
	cfg.DefaultAgent = a.Name
	if _, err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Printf("%s default agent: %s\n", ok(), bold(a.Name))
	return nil
}

func toggle(on bool) error {
	cfg, _, err := config.Load()
	if err != nil {
		return err
	}
	cfg.Enabled = on
	if _, err := config.Save(cfg); err != nil {
		return err
	}
	state := "off"
	if on {
		state = "on"
	}
	fmt.Printf("%s else.sh is %s\n", ok(), bold(state))
	return nil
}

func doctor() error {
	exe, _ := self()
	fmt.Printf("binary   %s (%s)\n", exe, Version)
	if sh, err := shell.Detect(); err != nil {
		fmt.Printf("shell    %s %v\n", warn(), err)
	} else if rc, hooked := shell.Installed(sh); hooked {
		fmt.Printf("shell    %s — hooked in %s\n", sh.Name, rc)
	} else {
		fmt.Printf("shell    %s — %s run `else.sh install`\n", sh.Name, warn())
	}
	path, _ := config.Path()
	cfg, exists, err := config.Load()
	switch {
	case err != nil:
		fmt.Printf("config   %s %v\n", warn(), err)
	case !exists:
		fmt.Printf("config   %s (not created yet)\n", path)
	default:
		fmt.Printf("config   %s\n", path)
	}
	fmt.Printf("enabled  %v, min_words %d, typo_guard %v\n", cfg.Enabled, cfg.MinWords, cfg.TypoGuard)
	if a, p, err := agent.Resolve(cfg); err != nil {
		fmt.Printf("agent    %s %v\n", warn(), err)
	} else {
		fmt.Printf("agent    %s (%s)\n", a.Name, p)
	}
	return nil
}

func agentNames(all []agent.Agent) string {
	n := make([]string, len(all))
	for i, a := range all {
		n[i] = a.Name
	}
	return strings.Join(n, ", ")
}

func self() (string, error) {
	// Symlinks are kept as-is so package-manager upgrades don't break the hook.
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exe)
}

var color = os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(os.Stderr.Fd()))

func paint(code, s string) string {
	if !color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func bold(s string) string  { return paint("1", s) }
func dim(s string) string   { return paint("2", s) }
func green(s string) string { return paint("32", s) }
func ok() string            { return paint("32", "✓") }
func warn() string          { return paint("33", "!") }
