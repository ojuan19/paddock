package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestLoad_NoFile(t *testing.T) {
	isolateHome(t)
	c, err := Load()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c == nil {
		t.Fatal("Load() returned nil config")
	}
	if c.Version != CurrentVersion {
		t.Fatalf("Version=%d want %d", c.Version, CurrentVersion)
	}
	if c.Profiles == nil {
		t.Fatal("Profiles is nil; want non-nil empty map")
	}
	if len(c.Profiles) != 0 {
		t.Fatalf("len(Profiles)=%d want 0", len(c.Profiles))
	}
}

func TestLoad_CorruptJSON(t *testing.T) {
	isolateHome(t)
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "parsing") && !strings.Contains(msg, "unmarshal") {
		t.Fatalf("err message %q lacks 'parsing' or 'unmarshal'", msg)
	}
}

func TestLoad_Valid(t *testing.T) {
	isolateHome(t)
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	raw := `{
  "version": 1,
  "default_profile": "work",
  "profiles": {
    "work": {"color": "blue", "created_at": "2025-01-01T00:00:00Z"},
    "personal": {"color": "green", "created_at": "2025-01-02T00:00:00Z"}
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if c.DefaultProfile != "work" {
		t.Fatalf("DefaultProfile=%q", c.DefaultProfile)
	}
	if len(c.Profiles) != 2 {
		t.Fatalf("len(Profiles)=%d want 2", len(c.Profiles))
	}
	if c.Profiles["work"].Color != "blue" {
		t.Fatalf("work color=%q", c.Profiles["work"].Color)
	}
	if c.Profiles["personal"].Color != "green" {
		t.Fatalf("personal color=%q", c.Profiles["personal"].Color)
	}
}

func TestSave_StampsCurrentVersion(t *testing.T) {
	isolateHome(t)
	c := &Config{Version: 0, Profiles: map[string]Profile{}}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Version != CurrentVersion {
		t.Fatalf("Version=%d want %d", loaded.Version, CurrentVersion)
	}
}

func TestSave_NilProfilesMap(t *testing.T) {
	isolateHome(t)
	c := &Config{Version: 1}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Profiles == nil {
		t.Fatal("Profiles is nil; want non-nil empty map")
	}
	if len(loaded.Profiles) != 0 {
		t.Fatalf("len(Profiles)=%d want 0", len(loaded.Profiles))
	}
}

func TestLastUsedAt_RoundTripNil(t *testing.T) {
	isolateHome(t)
	c := &Config{
		Version: 1,
		Profiles: map[string]Profile{
			"work": {Color: "blue", CreatedAt: time.Now().UTC(), LastUsedAt: nil},
		},
	}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "last_used_at") {
		t.Fatalf("on-disk JSON should omit last_used_at when nil; got:\n%s", data)
	}
}

func TestLastUsedAt_RoundTripSet(t *testing.T) {
	isolateHome(t)
	original := time.Now().UTC()
	c := &Config{
		Version: 1,
		Profiles: map[string]Profile{
			"work": {Color: "blue", CreatedAt: original, LastUsedAt: &original},
		},
	}
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := loaded.Profiles["work"].LastUsedAt
	if got == nil {
		t.Fatal("LastUsedAt is nil; want set")
	}
	if !got.Equal(original) {
		t.Fatalf("LastUsedAt=%v want %v", got, original)
	}
}

// Locks the Round 2 migration fix. If this regresses, statusline behavior and
// any time-based sort break silently.
func TestLastUsedAt_ZeroPointerNormalized(t *testing.T) {
	isolateHome(t)
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	raw := `{
  "version": 1,
  "profiles": {
    "work": {"color": "blue", "created_at": "2025-01-01T00:00:00Z", "last_used_at": "0001-01-01T00:00:00Z"}
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Profiles["work"].LastUsedAt != nil {
		t.Fatalf("LastUsedAt=%v want nil (zero-time should be normalized)", loaded.Profiles["work"].LastUsedAt)
	}
}

func TestAutoAssignColor(t *testing.T) {
	t.Run("empty_returns_blue", func(t *testing.T) {
		c := &Config{Profiles: map[string]Profile{}}
		if got := c.AutoAssignColor(); got != "blue" {
			t.Fatalf("got %q want blue", got)
		}
	})

	t.Run("four_used_returns_purple", func(t *testing.T) {
		c := &Config{Profiles: map[string]Profile{
			"a": {Color: "blue"},
			"b": {Color: "green"},
			"c": {Color: "amber"},
			"d": {Color: "red"},
		}}
		if got := c.AutoAssignColor(); got != "purple" {
			t.Fatalf("got %q want purple", got)
		}
	})

	// Spec lock: when all 9 colors are used, AutoAssignColor wraps via
	// AllowedColors[len(profiles) % len(AllowedColors)]. Reordering AllowedColors
	// is a breaking change to first-profile color assignment.
	t.Run("all_used_wraps_to_blue", func(t *testing.T) {
		profiles := map[string]Profile{}
		for i, color := range AllowedColors {
			profiles[string(rune('a'+i))] = Profile{Color: color}
		}
		c := &Config{Profiles: profiles}
		// 9 profiles, 9 colors, 9 % 9 == 0 -> AllowedColors[0] = "blue"
		if got := c.AutoAssignColor(); got != "blue" {
			t.Fatalf("got %q want blue (wraparound)", got)
		}
	})
}

func TestSentinelsAreDistinct(t *testing.T) {
	if errors.Is(ErrProfileExists, ErrProfileNotFound) {
		t.Fatal("ErrProfileExists must not be ErrProfileNotFound")
	}
}

func TestWriteFileAtomic(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	if err := WriteFileAtomic(path, []byte("hello")); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("got %q want hello", data)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf(".tmp file should not exist after rename; stat err=%v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("mode=%o want 0600", got)
		}
	}
}

func TestWriteFileAtomic_CreatesParentDir(t *testing.T) {
	isolateHome(t)
	base := t.TempDir()
	deep := filepath.Join(base, "a", "b")
	path := filepath.Join(deep, "file")
	if err := WriteFileAtomic(path, []byte("hi")); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(deep)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Fatalf("parent dir mode=%o want 0700", got)
		}
	}
}

func TestLinks_RoundTrip(t *testing.T) {
	isolateHome(t)
	in := &Links{Version: 1, Links: map[string]string{"/a/b": "foo", "/x": "bar"}}
	if err := in.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := LoadLinks()
	if err != nil {
		t.Fatalf("LoadLinks: %v", err)
	}
	if out.Version != CurrentVersion {
		t.Fatalf("Version=%d want %d", out.Version, CurrentVersion)
	}
	if len(out.Links) != 2 {
		t.Fatalf("len=%d want 2", len(out.Links))
	}
	if out.Links["/a/b"] != "foo" || out.Links["/x"] != "bar" {
		t.Fatalf("got %+v", out.Links)
	}
}

// Compile-time guard that encoding/json is reachable (test relies on it via
// the JSON file written by Save). Kept to keep the import used regardless of
// future test edits.
var _ = json.Marshal
