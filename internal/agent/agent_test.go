package agent

import (
	"slices"
	"testing"

	"github.com/cristi-boboc/else.sh/internal/config"
)

func TestArgv(t *testing.T) {
	a := Agent{Args: []string{"-i", Placeholder}}
	if got := a.Argv("hi there"); !slices.Equal(got, []string{"-i", "hi there"}) {
		t.Errorf("got %q", got)
	}
	b := Agent{Args: []string{"--yolo"}}
	if got := b.Argv("x"); !slices.Equal(got, []string{"--yolo", "x"}) {
		t.Errorf("appended prompt: got %q", got)
	}
}

func TestAllOverridesAndCustom(t *testing.T) {
	cfg := config.Default()
	cfg.Agents = map[string]config.Agent{
		"claude": {Args: []string{"--model", "opus", Placeholder}},
		"zeta":   {Command: "zeta-cli"},
	}
	all := All(cfg)
	c, _ := Find(all, "claude")
	if !slices.Equal(c.Args, []string{"--model", "opus", Placeholder}) || c.Command != "claude" {
		t.Errorf("override not applied: %+v", c)
	}
	z, ok := Find(all, "zeta")
	if !ok || z.Command != "zeta-cli" || all[len(all)-1].Name != "zeta" {
		t.Errorf("custom agent missing or misplaced: %+v", z)
	}
	if !slices.Equal(Builtins[0].Args, []string{Placeholder}) {
		t.Error("builtins mutated")
	}
}
