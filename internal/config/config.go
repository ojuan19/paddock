package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const CurrentVersion = 1

type Profile struct {
	Color      string    `json:"color"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
}

type Config struct {
	Version        int                `json:"version"`
	DefaultProfile string             `json:"default_profile,omitempty"`
	Profiles       map[string]Profile `json:"profiles"`
}

var AllowedColors = []string{"blue", "green", "amber", "red", "purple", "teal", "pink", "coral", "gray"}

var (
	ErrProfileExists   = errors.New("profile already exists")
	ErrProfileNotFound = errors.New("profile not found")
)

func Path() (string, error) {
	dir, err := paddockDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Version: CurrentVersion, Profiles: map[string]Profile{}}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	return &c, nil
}

func (c *Config) Save() error {
	// Always stamp the current schema version so migrated configs are normalized on write.
	c.Version = CurrentVersion
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	path, err := Path()
	if err != nil {
		return err
	}
	if err := writeFileAtomic(path, data); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	return nil
}

func (c *Config) AutoAssignColor() string {
	used := make(map[string]bool, len(c.Profiles))
	for _, p := range c.Profiles {
		used[p.Color] = true
	}
	for _, color := range AllowedColors {
		if !used[color] {
			return color
		}
	}
	return AllowedColors[len(c.Profiles)%len(AllowedColors)]
}
