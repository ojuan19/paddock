package shell

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestDetectShell_Zsh(t *testing.T) {
	home := setupHome(t)
	t.Setenv("SHELL", "/bin/zsh")
	name, rc, ok := DetectShell()
	if !ok || name != "zsh" {
		t.Fatalf("got name=%q ok=%v want zsh true", name, ok)
	}
	if rc != filepath.Join(home, ".zshrc") {
		t.Fatalf("rc=%q", rc)
	}
}

func TestDetectShell_Bash(t *testing.T) {
	home := setupHome(t)
	t.Setenv("SHELL", "/bin/bash")
	name, rc, ok := DetectShell()
	if !ok || name != "bash" {
		t.Fatalf("got name=%q ok=%v want bash true", name, ok)
	}
	if rc != filepath.Join(home, ".bashrc") {
		t.Fatalf("rc=%q", rc)
	}
}

func TestDetectShell_Fish(t *testing.T) {
	home := setupHome(t)
	t.Setenv("SHELL", "/usr/local/bin/fish")
	name, rc, ok := DetectShell()
	if !ok || name != "fish" {
		t.Fatalf("got name=%q ok=%v want fish true", name, ok)
	}
	if rc != filepath.Join(home, ".config", "fish", "config.fish") {
		t.Fatalf("rc=%q", rc)
	}
}

func TestDetectShell_Unsupported(t *testing.T) {
	setupHome(t)
	t.Setenv("SHELL", "/usr/bin/pwsh")
	name, rc, ok := DetectShell()
	if ok {
		t.Fatalf("expected unsupported, got name=%q rc=%q ok=true", name, rc)
	}
	if rc != "" {
		t.Fatalf("expected empty rc for unsupported, got %q", rc)
	}
}

func TestDetectShell_Empty(t *testing.T) {
	home := setupHome(t)
	t.Setenv("SHELL", "")
	name, rc, ok := DetectShell()
	if !ok || name != "zsh" {
		t.Fatalf("got name=%q ok=%v want zsh true (empty SHELL defaults to zsh)", name, ok)
	}
	if rc != filepath.Join(home, ".zshrc") {
		t.Fatalf("rc=%q", rc)
	}
}

func TestIsInstalled_FileMissing(t *testing.T) {
	home := setupHome(t)
	got, err := IsInstalled(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got {
		t.Fatal("expected false for missing file")
	}
}

func TestIsInstalled_MarkerAbsent(t *testing.T) {
	home := setupHome(t)
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("export FOO=bar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := IsInstalled(rc)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatal("expected false when marker absent")
	}
}

func TestIsInstalled_MarkerPresent(t *testing.T) {
	home := setupHome(t)
	rc := filepath.Join(home, ".zshrc")
	body := "export FOO=bar\n" + StartMarker + "\n# stuff\n" + EndMarker + "\n"
	if err := os.WriteFile(rc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := IsInstalled(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("expected true when marker present")
	}
}

func TestInstall_FreshFile(t *testing.T) {
	home := setupHome(t)
	rc := filepath.Join(home, ".zshrc")
	if err := Install("zsh", rc); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), StartMarker) {
		t.Fatalf("missing start marker in:\n%s", data)
	}
	script, _ := Script("zsh")
	if !strings.Contains(string(data), script) {
		t.Fatalf("rc does not contain the embedded script verbatim")
	}
}

func TestInstall_ExistingFile_AppendsBelow(t *testing.T) {
	home := setupHome(t)
	rc := filepath.Join(home, ".zshrc")
	prev := "# user content\nexport FOO=bar\n"
	if err := os.WriteFile(rc, []byte(prev), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Install("zsh", rc); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), prev) {
		t.Fatalf("user content not preserved at top:\n%s", data)
	}
	if !strings.Contains(string(data), StartMarker) {
		t.Fatalf("missing marker after append")
	}
}

func TestInstall_Idempotent(t *testing.T) {
	home := setupHome(t)
	rc := filepath.Join(home, ".zshrc")
	if err := Install("zsh", rc); err != nil {
		t.Fatal(err)
	}
	if err := Install("zsh", rc); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(data), StartMarker); count != 1 {
		t.Fatalf("expected 1 marker after two installs, got %d", count)
	}
}

func TestInstall_FishCreatesConfigDir(t *testing.T) {
	home := setupHome(t)
	rc := filepath.Join(home, ".config", "fish", "config.fish")
	if err := Install("fish", rc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(rc); err != nil {
		t.Fatalf("fish config not created: %v", err)
	}
	data, _ := os.ReadFile(rc)
	if !strings.Contains(string(data), StartMarker) {
		t.Fatalf("missing marker in fish config")
	}
}

func TestInstall_PreservesFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode semantics differ on windows")
	}
	home := setupHome(t)
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(rc, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Install("zsh", rc); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(rc)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("expected mode 0644 preserved, got %o", got)
	}
}

func TestInstall_Unsupported(t *testing.T) {
	setupHome(t)
	err := Install("pwsh", "")
	if err == nil {
		t.Fatal("expected error for unsupported shell")
	}
	if !errors.Is(err, ErrUnsupportedShell) {
		t.Fatalf("expected ErrUnsupportedShell, got %v", err)
	}
}
