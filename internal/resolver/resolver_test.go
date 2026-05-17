package resolver

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ojuan19/paddock/internal/config"
)

// isolateEnv clears resolver-influencing env vars for the test's duration.
func isolateEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PADDOCK_PROFILE", "")
}

// mkConfig builds a *config.Config with the given profiles + optional default.
func mkConfig(t *testing.T, defaultName string, profiles ...string) *config.Config {
	t.Helper()
	c := &config.Config{
		Version:        config.CurrentVersion,
		Profiles:       map[string]config.Profile{},
		DefaultProfile: defaultName,
	}
	for _, name := range profiles {
		c.Profiles[name] = config.Profile{Color: "blue", CreatedAt: time.Now().UTC()}
	}
	return c
}

// mkLinks builds a *config.Links from a map of path -> profile.
func mkLinks(m map[string]string) *config.Links {
	return &config.Links{Version: config.CurrentVersion, Links: m}
}

// writeDotPaddock writes content to <dir>/.paddock.
func writeDotPaddock(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".paddock"), []byte(content), 0o644); err != nil {
		t.Fatalf("writeDotPaddock: %v", err)
	}
}

func Test_Resolve_RuleEnvVar(t *testing.T) {
	t.Run("env_set_and_in_config", func(t *testing.T) {
		isolateEnv(t)
		t.Setenv("PADDOCK_PROFILE", "work")
		cfg := mkConfig(t, "", "work", "personal")
		res, err := Resolve(t.TempDir(), cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Profile != "work" || res.Rule != RuleEnvVar {
			t.Fatalf("got %+v, want profile=work rule=RuleEnvVar", res)
		}
	})

	t.Run("env_set_but_not_in_config", func(t *testing.T) {
		isolateEnv(t)
		t.Setenv("PADDOCK_PROFILE", "ghost")
		cfg := mkConfig(t, "", "work")
		_, err := Resolve(t.TempDir(), cfg, mkLinks(nil))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrProfileNotInConfig) {
			t.Fatalf("errors.Is(ErrProfileNotInConfig) = false, err = %v", err)
		}
		var nic *NotInConfigError
		if !errors.As(err, &nic) {
			t.Fatalf("errors.As(*NotInConfigError) = false, err = %v", err)
		}
		if nic.Name != "ghost" {
			t.Fatalf("nic.Name = %q, want ghost", nic.Name)
		}
		if nic.Source != "$PADDOCK_PROFILE" {
			t.Fatalf("nic.Source = %q, want $PADDOCK_PROFILE", nic.Source)
		}
	})

	t.Run("env_empty_falls_through", func(t *testing.T) {
		isolateEnv(t)
		t.Setenv("PADDOCK_PROFILE", "")
		cfg := mkConfig(t, "fallback", "fallback")
		res, err := Resolve(t.TempDir(), cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Rule != RuleDefaultProfile || res.Profile != "fallback" {
			t.Fatalf("got %+v, want profile=fallback rule=RuleDefaultProfile", res)
		}
	})

	t.Run("env_unset_falls_through", func(t *testing.T) {
		isolateEnv(t)
		os.Unsetenv("PADDOCK_PROFILE")
		cfg := mkConfig(t, "fallback", "fallback")
		res, err := Resolve(t.TempDir(), cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Rule != RuleDefaultProfile {
			t.Fatalf("got rule=%v, want RuleDefaultProfile", res.Rule)
		}
	})
}

func Test_Resolve_RuleDotPaddock(t *testing.T) {
	t.Run("at_pwd_directly", func(t *testing.T) {
		isolateEnv(t)
		home := os.Getenv("HOME")
		pwd := filepath.Join(home, "proj")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeDotPaddock(t, pwd, "work")
		cfg := mkConfig(t, "", "work")
		res, err := Resolve(pwd, cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Profile != "work" || res.Rule != RuleDotPaddockFile {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("at_parent", func(t *testing.T) {
		isolateEnv(t)
		home := os.Getenv("HOME")
		parent := filepath.Join(home, "proj")
		pwd := filepath.Join(parent, "sub")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeDotPaddock(t, parent, "work")
		cfg := mkConfig(t, "", "work")
		res, err := Resolve(pwd, cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Profile != "work" || res.Rule != RuleDotPaddockFile {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("at_grandparent", func(t *testing.T) {
		isolateEnv(t)
		home := os.Getenv("HOME")
		grand := filepath.Join(home, "proj")
		pwd := filepath.Join(grand, "a", "b")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeDotPaddock(t, grand, "work")
		cfg := mkConfig(t, "", "work")
		res, err := Resolve(pwd, cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Profile != "work" {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("at_home_boundary_inclusive", func(t *testing.T) {
		isolateEnv(t)
		home := os.Getenv("HOME")
		pwd := filepath.Join(home, "sub", "sub2")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeDotPaddock(t, home, "work")
		cfg := mkConfig(t, "", "work")
		res, err := Resolve(pwd, cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Profile != "work" || res.Rule != RuleDotPaddockFile {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("pwd_outside_home_does_not_match_home_paddock", func(t *testing.T) {
		isolateEnv(t)
		home := os.Getenv("HOME")
		writeDotPaddock(t, home, "work")
		// Use a sibling tempdir as pwd, outside HOME.
		outside := t.TempDir()
		pwd := filepath.Join(outside, "sub")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		cfg := mkConfig(t, "fallback", "work", "fallback")
		res, err := Resolve(pwd, cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		// Should fall through to default — walk doesn't cross into $HOME from outside.
		if res.Rule != RuleDefaultProfile {
			t.Fatalf("got rule=%v want RuleDefaultProfile (res=%+v)", res.Rule, res)
		}
	})

	t.Run("trailing_newline", func(t *testing.T) {
		isolateEnv(t)
		home := os.Getenv("HOME")
		pwd := filepath.Join(home, "p")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeDotPaddock(t, pwd, "work\n")
		cfg := mkConfig(t, "", "work")
		res, err := Resolve(pwd, cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Profile != "work" {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("crlf_line_ending", func(t *testing.T) {
		isolateEnv(t)
		home := os.Getenv("HOME")
		pwd := filepath.Join(home, "p")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeDotPaddock(t, pwd, "work\r\n")
		cfg := mkConfig(t, "", "work")
		res, err := Resolve(pwd, cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Profile != "work" {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("empty_file_falls_through", func(t *testing.T) {
		isolateEnv(t)
		home := os.Getenv("HOME")
		pwd := filepath.Join(home, "p")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeDotPaddock(t, pwd, "")
		cfg := mkConfig(t, "fallback", "fallback")
		res, err := Resolve(pwd, cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Rule != RuleDefaultProfile {
			t.Fatalf("got %+v, want RuleDefaultProfile", res)
		}
	})

	t.Run("whitespace_only_falls_through", func(t *testing.T) {
		isolateEnv(t)
		home := os.Getenv("HOME")
		pwd := filepath.Join(home, "p")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeDotPaddock(t, pwd, "   \t\n  ")
		cfg := mkConfig(t, "fallback", "fallback")
		res, err := Resolve(pwd, cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Rule != RuleDefaultProfile {
			t.Fatalf("got %+v, want RuleDefaultProfile", res)
		}
	})

	t.Run("invalid_name", func(t *testing.T) {
		isolateEnv(t)
		home := os.Getenv("HOME")
		pwd := filepath.Join(home, "p")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeDotPaddock(t, pwd, "../evil")
		cfg := mkConfig(t, "", "work")
		_, err := Resolve(pwd, cfg, mkLinks(nil))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrInvalidDotPaddock) {
			t.Fatalf("errors.Is(ErrInvalidDotPaddock) = false, err = %v", err)
		}
	})

	t.Run("valid_name_not_in_config", func(t *testing.T) {
		isolateEnv(t)
		home := os.Getenv("HOME")
		pwd := filepath.Join(home, "p")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeDotPaddock(t, pwd, "ghost")
		cfg := mkConfig(t, "", "work")
		_, err := Resolve(pwd, cfg, mkLinks(nil))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrProfileNotInConfig) {
			t.Fatalf("errors.Is(ErrProfileNotInConfig) = false, err = %v", err)
		}
		var nic *NotInConfigError
		if !errors.As(err, &nic) {
			t.Fatalf("errors.As(*NotInConfigError) = false, err = %v", err)
		}
		if nic.Name != "ghost" {
			t.Fatalf("nic.Name = %q want ghost", nic.Name)
		}
		if got := nic.Source; len(got) < len(".paddock file at") || got[:len(".paddock file at")] != ".paddock file at" {
			t.Fatalf("nic.Source = %q, want prefix '.paddock file at'", got)
		}
	})

	t.Run("home_unset_walks_to_root_no_panic", func(t *testing.T) {
		// With HOME unset, the walk falls to filesystem root without panic.
		// Place .paddock in a deep tempdir and verify it's found.
		t.Setenv("PADDOCK_PROFILE", "")
		// Build pwd FIRST while HOME is real so t.TempDir uses a valid base.
		base := t.TempDir()
		t.Setenv("HOME", "")
		pwd := filepath.Join(base, "a", "b")
		if err := os.MkdirAll(pwd, 0o755); err != nil {
			t.Fatal(err)
		}
		writeDotPaddock(t, base, "work")
		cfg := mkConfig(t, "", "work")
		res, err := Resolve(pwd, cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Profile != "work" || res.Rule != RuleDotPaddockFile {
			t.Fatalf("got %+v", res)
		}
	})
}

func Test_Resolve_RuleLinks(t *testing.T) {
	isolateEnv(t)
	tmp := t.TempDir()
	for _, p := range []string{
		filepath.Join(tmp, "a", "b", "c", "d"),
		filepath.Join(tmp, "x"),
		filepath.Join(tmp, "z"),
	} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	links := mkLinks(map[string]string{
		filepath.Join(tmp, "a"):           "p1",
		filepath.Join(tmp, "a", "b", "c"): "p2",
		filepath.Join(tmp, "x"):           "p3",
	})
	cfg := mkConfig(t, "pdefault", "p1", "p2", "p3", "pdefault")

	cases := []struct {
		name        string
		pwd         string
		wantProfile string
		wantRule    Rule
	}{
		{"longest_wins", filepath.Join(tmp, "a", "b", "c", "d"), "p2", RuleLinksJSON},
		{"parent_only", filepath.Join(tmp, "a", "b"), "p1", RuleLinksJSON},
		{"exact_match", filepath.Join(tmp, "x"), "p3", RuleLinksJSON},
		{"fallthrough", filepath.Join(tmp, "z"), "pdefault", RuleDefaultProfile},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			res, err := Resolve(tt.pwd, cfg, links)
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if res.Profile != tt.wantProfile {
				t.Fatalf("got profile=%q want %q", res.Profile, tt.wantProfile)
			}
			if res.Rule != tt.wantRule {
				t.Fatalf("got rule=%v want %v", res.Rule, tt.wantRule)
			}
		})
	}
}

func Test_Resolve_RuleDefault(t *testing.T) {
	t.Run("default_exists", func(t *testing.T) {
		isolateEnv(t)
		cfg := mkConfig(t, "main", "main")
		res, err := Resolve(t.TempDir(), cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Profile != "main" || res.Rule != RuleDefaultProfile {
			t.Fatalf("got %+v", res)
		}
	})

	t.Run("default_not_in_config", func(t *testing.T) {
		isolateEnv(t)
		cfg := mkConfig(t, "ghost") // no profiles
		_, err := Resolve(t.TempDir(), cfg, mkLinks(nil))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var nic *NotInConfigError
		if !errors.As(err, &nic) {
			t.Fatalf("errors.As(*NotInConfigError) = false, err = %v", err)
		}
		if nic.Source != "default_profile in config.json" {
			t.Fatalf("nic.Source = %q", nic.Source)
		}
	})

	t.Run("default_empty_no_other_rule", func(t *testing.T) {
		isolateEnv(t)
		cfg := mkConfig(t, "")
		res, err := Resolve(t.TempDir(), cfg, mkLinks(nil))
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		if res.Rule != RuleNone {
			t.Fatalf("got rule=%v want RuleNone", res.Rule)
		}
	})
}

func Test_Resolve_RuleNone(t *testing.T) {
	isolateEnv(t)
	cfg := mkConfig(t, "")
	res, err := Resolve(t.TempDir(), cfg, mkLinks(nil))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if res.Rule != RuleNone {
		t.Fatalf("got rule=%v want RuleNone", res.Rule)
	}
	if res.Profile != "" {
		t.Fatalf("got profile=%q want empty", res.Profile)
	}
}

func Test_Resolve_PwdNormalization(t *testing.T) {
	// Locks the Step 0 fix: filepath.Clean(pwd) at the top of Resolve.
	// Without it, trailing slashes or ./.. segments make filepath.Dir
	// spin one extra iteration from the wrong start, silently routing
	// to the wrong profile.
	cases := []struct {
		name string
		// build pwd as `base + suffix` after MkdirAll on the clean path.
		// .paddock will be written at the cleaned parent.
		mkPwd func(base string) string
	}{
		{"trailing_slash", func(base string) string {
			return filepath.Join(base, "a", "b") + string(filepath.Separator)
		}},
		{"dot_segment", func(base string) string {
			return filepath.Join(base, "a", ".", "b")
		}},
		{"dotdot_segment", func(base string) string {
			return filepath.Join(base, "a", "b", "..", "b")
		}},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			isolateEnv(t)
			home := os.Getenv("HOME")
			cleanPwd := filepath.Join(home, "a", "b")
			if err := os.MkdirAll(cleanPwd, 0o755); err != nil {
				t.Fatal(err)
			}
			parent := filepath.Join(home, "a")
			writeDotPaddock(t, parent, "work")
			cfg := mkConfig(t, "", "work")
			rawPwd := tt.mkPwd(home)
			res, err := Resolve(rawPwd, cfg, mkLinks(nil))
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			if res.Profile != "work" || res.Rule != RuleDotPaddockFile {
				t.Fatalf("rawPwd=%q got %+v want profile=work rule=RuleDotPaddockFile", rawPwd, res)
			}
		})
	}
}

// Round 2 design: resolver does NOT call filepath.EvalSymlinks. Symlinked pwd
// walks the link path, not the target. Changing this is a breaking change.
//
// To exercise this, we place .paddock at the *real target's parent* (outside HOME),
// then make the symlink live inside HOME. When the resolver walks via the link
// path, it walks the link's own parent chain (inside HOME), which never sees
// the .paddock — proving no EvalSymlinks. For control, when we call Resolve
// with the real path directly, we *also* don't find it because the real
// target's parent is outside HOME and the walk stops at the home boundary
// being inapplicable. So we use a second control: call Resolve with the real
// target's parent itself — that should NOT match either because the walk only
// reads .paddock at `cur` (the parent's own .paddock), and the parent IS
// outside HOME so homeStop=="" and the walk continues to root. We rely on
// the walk reading the .paddock at parent — that case DOES match.
func Test_Resolve_SymlinkSemantics(t *testing.T) {
	isolateEnv(t)
	home := os.Getenv("HOME")

	// Real target lives outside HOME.
	outside := t.TempDir()
	realParent := filepath.Join(outside, "realparent")
	realDir := filepath.Join(realParent, "realdir")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// .paddock lives at the real target's PARENT (not in realDir itself).
	writeDotPaddock(t, realParent, "foo")

	// Symlink inside HOME points to realDir.
	linkPath := filepath.Join(home, "link")
	if err := os.Symlink(realDir, linkPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	cfg := mkConfig(t, "fallback", "foo", "fallback")

	// Walk from the link path: parent chain is home/link -> home -> ...,
	// which never sees realParent/.paddock.
	res, err := Resolve(linkPath, cfg, mkLinks(nil))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if res.Rule == RuleDotPaddockFile {
		t.Fatalf("expected NOT to resolve via .paddock through symlink (Round 2 design: no EvalSymlinks), got %+v", res)
	}
	if res.Rule != RuleDefaultProfile {
		t.Fatalf("expected fall-through to default, got %+v", res)
	}

	// Control: walk from the *real* path. Walk chain realDir -> realParent
	// reads realParent/.paddock and matches.
	resCtrl, err := Resolve(realDir, cfg, mkLinks(nil))
	if err != nil {
		t.Fatalf("control: unexpected: %v", err)
	}
	if resCtrl.Rule != RuleDotPaddockFile || resCtrl.Profile != "foo" {
		t.Fatalf("control: expected .paddock match via real path, got %+v", resCtrl)
	}
}

func Test_Resolve_Concurrent(t *testing.T) {
	isolateEnv(t)
	home := os.Getenv("HOME")
	pwd := filepath.Join(home, "proj")
	if err := os.MkdirAll(pwd, 0o755); err != nil {
		t.Fatal(err)
	}
	writeDotPaddock(t, pwd, "work")
	cfg := mkConfig(t, "", "work")
	links := mkLinks(map[string]string{pwd: "work"})

	const N = 50
	var wg sync.WaitGroup
	results := make([]Result, N)
	errs := make([]error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = Resolve(pwd, cfg, links)
		}(i)
	}
	wg.Wait()
	for i := 0; i < N; i++ {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: unexpected err: %v", i, errs[i])
		}
		if results[i].Profile != "work" {
			t.Fatalf("goroutine %d: got profile=%q", i, results[i].Profile)
		}
	}
}
