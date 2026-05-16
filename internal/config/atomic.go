package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func paddockDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home dir: %w", err)
	}
	dir := filepath.Join(home, ".paddock")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("creating paddock dir: %w", err)
	}
	return dir, nil
}

// writeFileAtomic writes to <path>.tmp then renames so readers never see a partial file.
func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating parent dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}
