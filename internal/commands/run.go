package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/profile"
	"github.com/ojuan19/paddock/internal/resolver"
)

func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "run [args...]",
		Short:              "Resolve profile for current dir and launch claude",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRun(args)
		},
	}
	return cmd
}

func runRun(args []string) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found in PATH. Install Claude Code: https://claude.com/download")
	}

	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	links, err := config.LoadLinks()
	if err != nil {
		return fmt.Errorf("loading links: %w", err)
	}

	result, err := resolver.Resolve(pwd, cfg, links)
	if err != nil {
		return err
	}

	cmd := exec.Command(claudePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := os.Environ()

	if result.Rule != resolver.RuleNone {
		profileDir, err := profile.Dir(result.Profile)
		if err != nil {
			return fmt.Errorf("resolving profile dir: %w", err)
		}
		env = append(env, "CLAUDE_CONFIG_DIR="+profileDir)
		// Persist last-used BEFORE spawning so we still record activation even if the child
		// crashes or the user kills it; spawn failure leaves a harmless stale timestamp.
		cfg.TouchLastUsed(result.Profile)
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
	}
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
