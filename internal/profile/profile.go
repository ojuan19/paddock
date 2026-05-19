package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ojuan19/paddock/internal/config"
)

var nameRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var ErrInvalidName = errors.New("invalid profile name")

func ValidateName(name string) error {
	if name == "" || name == "." || name == ".." || !nameRe.MatchString(name) {
		return ErrInvalidName
	}
	return nil
}

func Dir(name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	path, err := config.Path()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), "profiles", name), nil
}

func Create(name string) (string, error) {
	dir, err := Dir(name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("creating profile dir: %w", err)
	}
	return dir, nil
}

type statuslineSettings struct {
	StatusLine struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		Padding int    `json:"padding"`
	} `json:"statusLine"`
}

// UpdateStatuslineCommand rewrites only the statusLine.{type,command,padding}
// keys of <profile-dir>/settings.json, preserving any other keys (hooks,
// permissions, etc.) the user or Claude may have added. Unlike
// WriteStatuslineSettings, this is non-destructive: callers that mutate an
// existing profile (e.g. rename) should use this.
func UpdateStatuslineCommand(name, color string) error {
	dir, err := Dir(name)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "settings.json")
	raw := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		// A corrupted existing file is tolerated by starting from an empty map;
		// this matches the spirit of WriteStatuslineSettings (which overwrites).
		_ = json.Unmarshal(data, &raw)
	}
	sl, _ := raw["statusLine"].(map[string]any)
	if sl == nil {
		sl = map[string]any{}
	}
	sl["type"] = "command"
	sl["command"] = fmt.Sprintf("paddock statusline --profile %s --color %s", name, color)
	if _, ok := sl["padding"]; !ok {
		sl["padding"] = 0
	}
	raw["statusLine"] = sl
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling statusline settings: %w", err)
	}
	return config.WriteFileAtomic(path, data)
}

// WriteStatuslineSettings writes <profile-dir>/settings.json telling Claude Code
// to invoke `paddock statusline` for its statusline bar. Idempotent — overwrites.
func WriteStatuslineSettings(name, color string) error {
	dir, err := Dir(name)
	if err != nil {
		return err
	}
	var s statuslineSettings
	s.StatusLine.Type = "command"
	s.StatusLine.Command = fmt.Sprintf("paddock statusline --profile %s --color %s", name, color)
	s.StatusLine.Padding = 0
	data, err := json.MarshalIndent(&s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling statusline settings: %w", err)
	}
	return config.WriteFileAtomic(filepath.Join(dir, "settings.json"), data)
}
