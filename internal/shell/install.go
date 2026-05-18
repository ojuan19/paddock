package shell

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ojuan19/paddock/internal/config"
)

// DetectShell reads $SHELL and returns the shell name, the resolved rc path,
// and a supported flag. When supported is false the caller should skip the
// install prompt entirely (e.g. powershell, unknown shells).
// An empty $SHELL defaults to zsh, mirroring the rest of paddock.
func DetectShell() (name, rcPath string, supported bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", false
	}
	raw := os.Getenv("SHELL")
	sh := filepath.Base(raw)
	if raw == "" || sh == "." || sh == "/" {
		sh = "zsh"
	}
	switch sh {
	case "zsh":
		return "zsh", filepath.Join(home, ".zshrc"), true
	case "bash":
		return "bash", filepath.Join(home, ".bashrc"), true
	case "fish":
		return "fish", filepath.Join(home, ".config", "fish", "config.fish"), true
	default:
		return sh, "", false
	}
}

// IsInstalled returns true when the rc file contains the paddock StartMarker.
// A missing rc file returns (false, nil) — that's the "needs install" state,
// not an error.
func IsInstalled(rcPath string) (bool, error) {
	data, err := os.ReadFile(rcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return strings.Contains(string(data), StartMarker), nil
}

// Install appends the shell integration script (Script(name)) to rcPath,
// idempotently. If the file already contains StartMarker, this is a no-op.
// Parent directories and the rc file itself are created if missing.
// When the rc file exists, its mode is preserved across the atomic write
// (config.WriteFileAtomic creates files at 0o600; we restore the prior mode).
func Install(name, rcPath string) error {
	script, err := Script(name)
	if err != nil {
		return err
	}
	if rcPath == "" {
		return ErrUnsupportedShell
	}

	var (
		existing []byte
		prevMode os.FileMode
		hadFile  bool
	)
	if info, statErr := os.Stat(rcPath); statErr == nil {
		hadFile = true
		prevMode = info.Mode().Perm()
		existing, err = os.ReadFile(rcPath)
		if err != nil {
			return err
		}
		if strings.Contains(string(existing), StartMarker) {
			return nil
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}

	var buf strings.Builder
	if len(existing) > 0 {
		buf.Write(existing)
		if !strings.HasSuffix(string(existing), "\n") {
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}
	buf.WriteString(script)
	if !strings.HasSuffix(script, "\n") {
		buf.WriteString("\n")
	}

	if err := config.WriteFileAtomic(rcPath, []byte(buf.String())); err != nil {
		return err
	}
	if hadFile && prevMode != 0 {
		if err := os.Chmod(rcPath, prevMode); err != nil {
			return err
		}
	}
	return nil
}
