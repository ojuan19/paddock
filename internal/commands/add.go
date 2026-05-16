package commands

import (
	"fmt"
	"slices"
	"time"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/profile"
)

func NewAddCmd() *cobra.Command {
	var colorFlag string
	var defaultFlag bool

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := profile.ValidateName(name); err != nil {
				return fmt.Errorf("%w: %q (allowed: letters, digits, '-', '_')", err, name)
			}

			c, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if _, ok := c.Profiles[name]; ok {
				return fmt.Errorf("%w: %q", config.ErrProfileExists, name)
			}

			var color string
			if colorFlag != "" {
				if !slices.Contains(config.AllowedColors, colorFlag) {
					return fmt.Errorf("invalid color %q (allowed: %v)", colorFlag, config.AllowedColors)
				}
				color = colorFlag
			} else {
				color = c.AutoAssignColor()
			}

			if _, err := profile.Create(name); err != nil {
				return fmt.Errorf("creating profile dir: %w", err)
			}

			c.Profiles[name] = config.Profile{
				Color:     color,
				CreatedAt: time.Now().UTC(),
			}

			if defaultFlag || len(c.Profiles) == 1 {
				c.DefaultProfile = name
			}

			if err := c.Save(); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Created profile '%s' (%s)\n", name, color)
			return nil
		},
	}

	cmd.Flags().StringVar(&colorFlag, "color", "", "color label for the profile")
	cmd.Flags().BoolVar(&defaultFlag, "default", false, "set as the default profile")
	return cmd
}
