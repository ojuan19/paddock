package commands

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/profile"
	"github.com/ojuan19/paddock/internal/ui"
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
				return errors.New(ui.Error(fmt.Sprintf("Profile %q already exists", name)))
			}

			var color string
			if colorFlag != "" {
				if !slices.Contains(config.AllowedColors, colorFlag) {
					msg := ui.Error(fmt.Sprintf("Invalid color %q", colorFlag)) +
						"\n" + ui.Muted("  Allowed: "+strings.Join(config.AllowedColors, ", "))
					return errors.New(msg)
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

			isDefault := defaultFlag || len(c.Profiles) == 1
			if isDefault {
				c.DefaultProfile = name
			}

			if err := c.Save(); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			suffix := color
			if isDefault {
				suffix = color + ", default"
			}
			line := ui.Success("Created profile ") + ui.ProfileDot(color) + " " + ui.ProfileName(color, name) +
				"  " + ui.Muted("("+suffix+")")
			fmt.Fprintln(cmd.OutOrStdout(), line)
			return nil
		},
	}

	cmd.Flags().StringVar(&colorFlag, "color", "", "color label for the profile")
	cmd.Flags().BoolVar(&defaultFlag, "default", false, "set as the default profile")
	return cmd
}
