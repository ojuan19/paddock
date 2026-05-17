package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/ui"
)

func NewUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "use <profile> [args...]",
		Short:              "Launch claude using the specified profile",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("profile name required: paddock use <profile> [args...]")
			}
			name := args[0]
			claudeArgs := args[1:]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			if _, ok := cfg.Profiles[name]; !ok {
				msg := ui.Error(fmt.Sprintf("No profile named %q", name))
				if s := ui.SuggestProfile(name, profileNames(cfg)); s != "" {
					msg += "\n" + s
				}
				return errors.New(strings.TrimRight(msg, "\n"))
			}
			return spawnClaude(cfg, name, claudeArgs)
		},
	}
	return cmd
}
