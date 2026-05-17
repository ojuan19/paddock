package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/resolver"
	"github.com/ojuan19/paddock/internal/ui"
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
		var nice *resolver.NotInConfigError
		if errors.As(err, &nice) {
			msg := ui.Error(nice.Error())
			if s := ui.SuggestProfile(nice.Name, profileNames(cfg)); s != "" {
				msg += "\n" + s
			}
			return errors.New(msg)
		}
		return err
	}

	if result.Rule == resolver.RuleNone {
		return spawnClaude(cfg, "", args)
	}
	return spawnClaude(cfg, result.Profile, args)
}
