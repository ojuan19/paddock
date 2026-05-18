package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/shell"
)

// seedProfile prepares a HOME with a single registered profile and runs the
// caller in a fresh tempdir so `paddock link` has a target for the link.
func seedProfile(t *testing.T, name string) (home, workDir string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			name: {Color: "blue", CreatedAt: time.Now().UTC()},
		},
		DefaultProfile: name,
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("seeding config: %v", err)
	}

	workDir = t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	return home, workDir
}

func runLinkCmd(t *testing.T, stdin io.Reader, args ...string) (string, error) {
	t.Helper()
	cmd := NewLinkCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	cmd.SetIn(stdin)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestLink_InstallsHookWithYes(t *testing.T) {
	home, _ := seedProfile(t, "p1")
	rc := filepath.Join(home, ".zshrc")
	if _, err := os.Stat(rc); err == nil {
		t.Fatalf("precondition: rc should not exist yet")
	}

	prev := isStdinTTY
	isStdinTTY = func() bool { return false }
	t.Cleanup(func() { isStdinTTY = prev })

	out, err := runLinkCmd(t, nil, "p1", "--yes")
	if err != nil {
		t.Fatalf("link --yes: %v\n%s", err, out)
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("rc not created: %v", err)
	}
	if !strings.Contains(string(data), shell.StartMarker) {
		t.Fatalf("rc missing marker:\n%s", data)
	}
	if !strings.Contains(out, "Added paddock hook to") {
		t.Fatalf("expected install confirmation in output; got:\n%s", out)
	}
}

func TestLink_NoInstallWhenNotTTY(t *testing.T) {
	home, _ := seedProfile(t, "p1")
	rc := filepath.Join(home, ".zshrc")

	prev := isStdinTTY
	isStdinTTY = func() bool { return false }
	t.Cleanup(func() { isStdinTTY = prev })

	out, err := runLinkCmd(t, nil, "p1")
	if err != nil {
		t.Fatalf("link: %v\n%s", err, out)
	}
	if _, err := os.Stat(rc); !os.IsNotExist(err) {
		t.Fatalf("rc should not exist after non-TTY link without --yes (err=%v)", err)
	}
	if strings.Contains(out, "Shell auto-switch not installed") {
		t.Fatalf("did not expect prompt in non-TTY mode; got:\n%s", out)
	}
}

func TestLink_NoPromptWhenAlreadyInstalled(t *testing.T) {
	home, _ := seedProfile(t, "p1")
	rc := filepath.Join(home, ".zshrc")
	preexisting := "# user content\n" + shell.StartMarker + "\n# body\n" + shell.EndMarker + "\n"
	if err := os.WriteFile(rc, []byte(preexisting), 0o644); err != nil {
		t.Fatal(err)
	}

	prev := isStdinTTY
	isStdinTTY = func() bool { return true }
	t.Cleanup(func() { isStdinTTY = prev })

	out, err := runLinkCmd(t, nil, "p1")
	if err != nil {
		t.Fatalf("link: %v\n%s", err, out)
	}
	if strings.Contains(out, "Shell auto-switch not installed") {
		t.Fatalf("did not expect prompt when already installed; got:\n%s", out)
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != preexisting {
		t.Fatalf("rc was modified; want unchanged\nbefore:\n%s\nafter:\n%s", preexisting, data)
	}
}

func TestLink_DeclinedPromptKeepsFileUnmodified(t *testing.T) {
	home, _ := seedProfile(t, "p1")
	rc := filepath.Join(home, ".zshrc")
	preexisting := "# user content only\n"
	if err := os.WriteFile(rc, []byte(preexisting), 0o644); err != nil {
		t.Fatal(err)
	}

	prev := isStdinTTY
	isStdinTTY = func() bool { return true }
	t.Cleanup(func() { isStdinTTY = prev })

	out, err := runLinkCmd(t, strings.NewReader("n\n"), "p1")
	if err != nil {
		t.Fatalf("link: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Skipped. To install later") {
		t.Fatalf("expected decline message; got:\n%s", out)
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != preexisting {
		t.Fatalf("rc was modified after decline; want unchanged\nbefore:\n%s\nafter:\n%s", preexisting, data)
	}
}
