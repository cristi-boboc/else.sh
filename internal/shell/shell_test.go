package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{"-zsh": "zsh", "/bin/bash": "bash", "pwsh.exe": "powershell", "PowerShell.exe": "powershell", "fish": "fish"} {
		if got, ok := Normalize(in); !ok || got != want {
			t.Errorf("Normalize(%q) = %q, %v", in, got, ok)
		}
	}
	if _, ok := Normalize("cmd.exe"); ok {
		t.Error("cmd.exe should be unsupported")
	}
}

func TestInitQuotesExe(t *testing.T) {
	exe := "/opt/it's here/else.sh"
	for _, name := range Names {
		s, err := Init(name, exe, true)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(s, "__EXE__") || strings.Contains(s, "__NOMATCH__") {
			t.Errorf("%s: unreplaced placeholder", name)
		}
	}
	z, _ := Init(Zsh, exe, true)
	if !strings.Contains(z, "setopt no_nomatch") || !strings.Contains(z, `'/opt/it'\''s here/else.sh'`) {
		t.Errorf("zsh script wrong:\n%s", z)
	}
	if p, _ := Init(PowerShell, exe, true); !strings.Contains(p, `'/opt/it''s here/else.sh'`) {
		t.Errorf("powershell quoting wrong:\n%s", p)
	}
}

func TestInstallIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("ZDOTDIR", "")
	rc := filepath.Join(home, ".zshrc")
	os.WriteFile(rc, []byte("export FOO=1"), 0o644)
	sh := Shell{Name: Zsh}

	if _, changed, err := Install(sh, "/a/else.sh"); err != nil || !changed {
		t.Fatalf("first install: changed=%v err=%v", changed, err)
	}
	if _, changed, _ := Install(sh, "/a/else.sh"); changed {
		t.Error("second install should be a no-op")
	}
	Install(sh, "/b/else.sh")
	data, _ := os.ReadFile(rc)
	if strings.Count(string(data), beginMarker) != 1 || !strings.Contains(string(data), "/b/else.sh") {
		t.Errorf("block not replaced:\n%s", data)
	}
	if _, changed, _ := Uninstall(sh); !changed {
		t.Error("uninstall did nothing")
	}
	data, _ = os.ReadFile(rc)
	if strings.TrimSpace(string(data)) != "export FOO=1" {
		t.Errorf("uninstall left %q", data)
	}
}
