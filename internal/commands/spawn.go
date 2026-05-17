// spawnClaude is the shared claude-launch primitive. Empty profileName = no CLAUDE_CONFIG_DIR override.
package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/profile"
	"github.com/ojuan19/paddock/internal/ui"
)

func spawnClaude(cfg *config.Config, profileName string, args []string) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		msg := ui.Error("claude not found in PATH") +
			"\n" + ui.Muted("  Install Claude Code: https://claude.com/download")
		return errors.New(msg)
	}

	env := os.Environ()
	if profileName != "" {
		if _, ok := cfg.Profiles[profileName]; !ok {
			return fmt.Errorf("profile %q is not registered", profileName)
		}
		profileDir, err := profile.Dir(profileName)
		if err != nil {
			return fmt.Errorf("resolving profile dir: %w", err)
		}
		if info, err := os.Stat(profileDir); err != nil || !info.IsDir() {
			return fmt.Errorf("profile %q is registered but its directory is missing; recreate with paddock add", profileName)
		}
		env = append(env, "CLAUDE_CONFIG_DIR="+profileDir)
		// Persist last-used BEFORE spawning so we still record activation even if the child
		// crashes or the user kills it; spawn failure leaves a harmless stale timestamp.
		cfg.TouchLastUsed(profileName)
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
	}

	cmd := exec.Command(claudePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			os.Exit(ee.ExitCode())
		}
		return fmt.Errorf("running claude: %w", err)
	}
	return nil
}
