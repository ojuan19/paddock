package shell

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestScriptUnsupportedShell(t *testing.T) {
	_, err := Script("tcsh")
	if !errors.Is(err, ErrUnsupportedShell) {
		t.Fatalf("expected ErrUnsupportedShell, got %v", err)
	}
}

func TestScriptContents(t *testing.T) {
	requiredTokens := []string{
		"paddock which --quiet",
		"CLAUDE_CONFIG_DIR",
		StartMarker,
		EndMarker,
	}
	requiredAliasOrFunc := map[string]string{
		"bash":       "alias claude='paddock run'",
		"zsh":        "alias claude='paddock run'",
		"fish":       "paddock run $argv",
		"powershell": "paddock run @args",
	}
	for _, name := range []string{"bash", "zsh", "fish", "powershell"} {
		script, err := Script(name)
		if err != nil {
			t.Fatalf("Script(%q): %v", name, err)
		}
		if script == "" {
			t.Fatalf("Script(%q) is empty", name)
		}
		for _, tok := range requiredTokens {
			if !strings.Contains(script, tok) {
				t.Errorf("Script(%q) missing token %q", name, tok)
			}
		}
		if want := requiredAliasOrFunc[name]; !strings.Contains(script, want) {
			t.Errorf("Script(%q) missing claude wrapper %q", name, want)
		}
	}
}

func TestScriptSyntax(t *testing.T) {
	cases := []struct {
		name   string
		binary string
		flag   string
	}{
		{"bash", "bash", "-n"},
		{"zsh", "zsh", "-n"},
		// fish has no -n; use --no-execute via -n flag in fish 3+
		{"fish", "fish", "-n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := exec.LookPath(c.binary); err != nil {
				t.Skipf("%s not installed", c.binary)
			}
			script, err := Script(c.name)
			if err != nil {
				t.Fatalf("Script(%q): %v", c.name, err)
			}
			tmp := filepath.Join(t.TempDir(), c.name+".sh")
			if err := os.WriteFile(tmp, []byte(script), 0o644); err != nil {
				t.Fatalf("write tmp: %v", err)
			}
			cmd := exec.Command(c.binary, c.flag, tmp)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s %s %s failed: %v\n%s", c.binary, c.flag, tmp, err, out)
			}
		})
	}
}
