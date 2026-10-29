// Package config loads and saves the else.sh configuration file.
package config

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

// Agent is a user-defined (or overridden) agent entry.
type Agent struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

// Config is the on-disk configuration.
type Config struct {
	// Enabled turns the unknown-command hand-off on or off.
	Enabled bool `toml:"enabled"`
	// DefaultAgent is the agent that receives prompts.
	DefaultAgent string `toml:"default_agent"`
	// MinWords is the minimum number of words before a line is treated as a prompt.
	MinWords int `toml:"min_words"`
	// TypoGuard skips lines whose first word is one edit away from a real command.
	TypoGuard bool `toml:"typo_guard"`
	// ZshNoNomatch sets `setopt no_nomatch` so `?` and `*` in prompts don't abort the line.
	ZshNoNomatch bool `toml:"zsh_no_nomatch"`
	// Announce prints a one-line notice before the agent starts.
	Announce bool `toml:"announce"`
	// Agents adds new agents or overrides built-in ones.
	Agents map[string]Agent `toml:"agents"`
}

// Default returns the configuration used when no file exists.
func Default() Config {
	return Config{
		Enabled:      true,
		MinWords:     2,
		TypoGuard:    true,
		ZshNoNomatch: true,
		Announce:     true,
	}
}

// Path returns the config file location. ELSE_SH_CONFIG overrides it.
func Path() (string, error) {
	if p := os.Getenv("ELSE_SH_CONFIG"); p != "" {
		return p, nil
	}
	if runtime.GOOS == "windows" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "else.sh", "config.toml"), nil
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "else.sh", "config.toml"), nil
}

// Load reads the config file. The boolean reports whether the file existed.
func Load() (Config, bool, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, false, nil
	}
	if err != nil {
		return cfg, false, err
	}
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return Default(), true, err
	}
	if cfg.MinWords < 1 {
		cfg.MinWords = 1
	}
	return cfg, true, nil
}

const header = `# else.sh configuration — https://cristi-boboc.github.io/else.sh/
#
# Add your own agent (or override a built-in one); "{prompt}" is replaced
# with what you typed:
#
#   [agents.myagent]
#   command = "myagent"
#   args = ["--interactive", "{prompt}"]

`

// Save writes the config file, creating its directory.
func Save(cfg Config) (string, error) {
	path, err := Path()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	buf.WriteString(header)
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, buf.Bytes(), 0o644)
}
