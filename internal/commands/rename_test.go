package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/profile"
)

// seedForRename builds a HOME with the given profile(s) registered and their
// directories created on disk (with the default settings.json written). The
// first profile becomes DefaultProfile.
func seedForRename(t *testing.T, names ...string) string {
	t.Helper()
	if len(names) == 0 {
		t.Fatal("seedForRename: need at least one name")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &config.Config{
		Profiles:       map[string]config.Profile{},
		DefaultProfile: names[0],
	}
	for i, n := range names {
		cfg.Profiles[n] = config.Profile{
			Color:     []string{"blue", "green", "amber", "red"}[i%4],
			CreatedAt: time.Now().UTC(),
		}
		if _, err := profile.Create(n); err != nil {
			t.Fatalf("creating profile dir for %q: %v", n, err)
		}
		if err := profile.WriteStatuslineSettings(n, cfg.Profiles[n].Color); err != nil {
			t.Fatalf("writing statusline settings for %q: %v", n, err)
		}
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("saving cfg: %v", err)
	}
	return home
}

func runRenameCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRenameCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestRename_HappyPath(t *testing.T) {
	home := seedForRename(t, "work")

	out, err := runRenameCmd(t, "work", "consulting")
	if err != nil {
		t.Fatalf("rename: %v\n%s", err, out)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Profiles["work"]; ok {
		t.Fatal("old name still in config")
	}
	if _, ok := cfg.Profiles["consulting"]; !ok {
		t.Fatal("new name missing in config")
	}
	if cfg.DefaultProfile != "consulting" {
		t.Fatalf("DefaultProfile=%q want consulting", cfg.DefaultProfile)
	}

	// Old dir gone, new dir present.
	if _, err := os.Stat(filepath.Join(home, ".paddock", "profiles", "work")); !os.IsNotExist(err) {
		t.Fatalf("old dir still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".paddock", "profiles", "consulting")); err != nil {
		t.Fatalf("new dir missing: %v", err)
	}

	// settings.json command updated.
	data, err := os.ReadFile(filepath.Join(home, ".paddock", "profiles", "consulting", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "--profile consulting") {
		t.Fatalf("settings.json missing new profile name:\n%s", string(data))
	}
	if strings.Contains(string(data), "--profile work") {
		t.Fatalf("settings.json still references old name:\n%s", string(data))
	}
}

func TestRename_NotDefault_PreservesDefault(t *testing.T) {
	seedForRename(t, "personal", "work") // personal is default

	if _, err := runRenameCmd(t, "work", "consulting"); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load()
	if cfg.DefaultProfile != "personal" {
		t.Fatalf("DefaultProfile=%q want personal", cfg.DefaultProfile)
	}
}

func TestRename_UpdatesLinks(t *testing.T) {
	seedForRename(t, "work")
	links := &config.Links{Links: map[string]string{
		"/tmp/repo-a": "work",
		"/tmp/repo-b": "work",
		"/tmp/repo-c": "other",
	}}
	if err := links.Save(); err != nil {
		t.Fatal(err)
	}

	if _, err := runRenameCmd(t, "work", "consulting"); err != nil {
		t.Fatal(err)
	}
	got, _ := config.LoadLinks()
	if got.Links["/tmp/repo-a"] != "consulting" {
		t.Fatalf("repo-a=%q", got.Links["/tmp/repo-a"])
	}
	if got.Links["/tmp/repo-b"] != "consulting" {
		t.Fatalf("repo-b=%q", got.Links["/tmp/repo-b"])
	}
	if got.Links["/tmp/repo-c"] != "other" {
		t.Fatalf("repo-c got rewritten: %q", got.Links["/tmp/repo-c"])
	}
}

func TestRename_RewritesDotPaddockFiles(t *testing.T) {
	seedForRename(t, "work")

	linkedA := t.TempDir()
	linkedB := t.TempDir()
	linkedC := t.TempDir()
	if err := os.WriteFile(filepath.Join(linkedA, ".paddock"), []byte("work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// linkedB: linked in links.json but no .paddock file on disk.
	// linkedC: contains a .paddock pointing to a different name; must be left alone.
	if err := os.WriteFile(filepath.Join(linkedC, ".paddock"), []byte("other\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	links := &config.Links{Links: map[string]string{
		linkedA: "work",
		linkedB: "work",
		linkedC: "work", // links.json points work, but local .paddock disagrees (corner case)
	}}
	if err := links.Save(); err != nil {
		t.Fatal(err)
	}

	if _, err := runRenameCmd(t, "work", "consulting"); err != nil {
		t.Fatal(err)
	}

	a, _ := os.ReadFile(filepath.Join(linkedA, ".paddock"))
	if strings.TrimSpace(string(a)) != "consulting" {
		t.Fatalf("linkedA .paddock=%q", string(a))
	}
	if _, err := os.Stat(filepath.Join(linkedB, ".paddock")); !os.IsNotExist(err) {
		t.Fatalf("linkedB .paddock unexpectedly exists")
	}
	c, _ := os.ReadFile(filepath.Join(linkedC, ".paddock"))
	if strings.TrimSpace(string(c)) != "other" {
		t.Fatalf("linkedC .paddock got rewritten: %q", string(c))
	}
}

func TestRename_TargetExists(t *testing.T) {
	home := seedForRename(t, "work", "personal")

	_, err := runRenameCmd(t, "work", "personal")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err=%v", err)
	}
	// Zero mutations: both profiles still present, dirs intact.
	cfg, _ := config.Load()
	if _, ok := cfg.Profiles["work"]; !ok {
		t.Fatal("work was mutated")
	}
	if _, ok := cfg.Profiles["personal"]; !ok {
		t.Fatal("personal was mutated")
	}
	if _, err := os.Stat(filepath.Join(home, ".paddock", "profiles", "work")); err != nil {
		t.Fatalf("work dir gone: %v", err)
	}
}

func TestRename_SourceMissing(t *testing.T) {
	seedForRename(t, "work")
	_, err := runRenameCmd(t, "nonesuch", "consulting")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "No profile named") {
		t.Fatalf("err=%v", err)
	}
}

func TestRename_InvalidNewName(t *testing.T) {
	seedForRename(t, "work")
	for _, bad := range []string{"", ".", "..", "with space", "with/slash"} {
		_, err := runRenameCmd(t, "work", bad)
		if err == nil {
			t.Fatalf("name %q: expected error", bad)
		}
	}
}

func TestRename_SameName(t *testing.T) {
	seedForRename(t, "work")
	_, err := runRenameCmd(t, "work", "work")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRename_OrphanTargetDir(t *testing.T) {
	home := seedForRename(t, "work")
	// Pre-create an orphan target dir (not in config) — rename should refuse
	// rather than blast over it.
	orphan := filepath.Join(home, ".paddock", "profiles", "consulting")
	if err := os.MkdirAll(orphan, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphan, "marker"), []byte("dont_lose_me"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := runRenameCmd(t, "work", "consulting")
	if err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(filepath.Join(orphan, "marker")); err != nil {
		t.Fatalf("orphan marker lost: %v", err)
	}
}

func TestRename_PreservesUnknownSettingsKeys(t *testing.T) {
	home := seedForRename(t, "work")
	// Inject extra keys into the existing settings.json.
	path := filepath.Join(home, ".paddock", "profiles", "work", "settings.json")
	raw := map[string]any{}
	data, _ := os.ReadFile(path)
	_ = json.Unmarshal(data, &raw)
	raw["hooks"] = map[string]any{"PreToolUse": []any{"echo hi"}}
	raw["permissions"] = map[string]any{"allow": []any{"Bash(ls:*)"}}
	out, _ := json.MarshalIndent(raw, "", "  ")
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := runRenameCmd(t, "work", "consulting"); err != nil {
		t.Fatal(err)
	}

	newPath := filepath.Join(home, ".paddock", "profiles", "consulting", "settings.json")
	final := map[string]any{}
	d, _ := os.ReadFile(newPath)
	if err := json.Unmarshal(d, &final); err != nil {
		t.Fatalf("unmarshaling final settings.json: %v\n%s", err, string(d))
	}
	if _, ok := final["hooks"]; !ok {
		t.Fatalf("hooks key lost:\n%s", string(d))
	}
	if _, ok := final["permissions"]; !ok {
		t.Fatalf("permissions key lost:\n%s", string(d))
	}
	sl, _ := final["statusLine"].(map[string]any)
	cmdStr, _ := sl["command"].(string)
	if !strings.Contains(cmdStr, "--profile consulting") {
		t.Fatalf("statusLine.command not updated: %v", cmdStr)
	}
}

// TestRename_RollbackOnLinksFailure simulates a Save failure mid-rename by
// chmod'ing the .paddock dir read-only after the dir/config writes succeed.
// Expectation: dir is renamed back, config is reverted, no half-state.
func TestRename_RollbackOnLinksFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("can't usefully test chmod-based failure as root")
	}
	home := seedForRename(t, "work")
	// Force a links.json that will need writing.
	links := &config.Links{Links: map[string]string{"/tmp/repo-a": "work"}}
	if err := links.Save(); err != nil {
		t.Fatal(err)
	}

	// Make ~/.paddock read-only AFTER config.json is in place so that the
	// first writes succeed but the *next* save (links.json) fails.
	paddockDir := filepath.Join(home, ".paddock")
	// Read-only on the dir blocks WriteFileAtomic's rename step. But we also
	// need config.Save() to succeed earlier... config.Save also writes into
	// ~/.paddock. So this strategy blocks too early.
	//
	// Trick: shadow links.json with a directory of the same name so that the
	// rename(tmp → links.json) step fails specifically.
	linksPath := filepath.Join(paddockDir, "links.json")
	if err := os.Remove(linksPath); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(linksPath, 0o700); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(linksPath)

	_, err := runRenameCmd(t, "work", "consulting")
	if err == nil {
		t.Fatal("expected error from links save")
	}

	// After rollback: profile is named "work" again, dir is "work", config OK.
	cfg, loadErr := config.Load()
	if loadErr != nil {
		t.Fatalf("loading config after rollback: %v", loadErr)
	}
	if _, ok := cfg.Profiles["work"]; !ok {
		t.Fatalf("rollback failed: %q missing", "work")
	}
	if _, ok := cfg.Profiles["consulting"]; ok {
		t.Fatal("rollback failed: consulting still in config")
	}
	if _, statErr := os.Stat(filepath.Join(paddockDir, "profiles", "work")); statErr != nil {
		t.Fatalf("rollback failed: work dir missing: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(paddockDir, "profiles", "consulting")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("rollback failed: consulting dir still exists")
	}
}

// quick io.Reader sanity to keep the linter happy if we ever wire stdin tests.
var _ io.Reader = strings.NewReader("")
