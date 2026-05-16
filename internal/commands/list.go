package commands

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/ui"
)

func NewListCmd() *cobra.Command {
	var showLinks bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List profiles",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			links, err := config.LoadLinks()
			if err != nil {
				return fmt.Errorf("loading links: %w", err)
			}

			out := cmd.OutOrStdout()
			if len(c.Profiles) == 0 {
				fmt.Fprintln(out, ui.Muted("No profiles yet. Run '")+ui.Bold("paddock add <name>")+ui.Muted("'."))
				return nil
			}

			names := profileNames(c)
			maxLen := 0
			for _, n := range names {
				if len(n) > maxLen {
					maxLen = len(n)
				}
			}

			for _, name := range names {
				count := 0
				var dirs []string
				for dir, p := range links.Links {
					if p == name {
						count++
						if showLinks {
							dirs = append(dirs, dir)
						}
					}
				}
				p := c.Profiles[name]
				padded := fmt.Sprintf("%-*s", maxLen, name)
				dot := ui.ProfileDot(p.Color)
				prefix := ""
				if dot != "" {
					prefix = dot + " "
				}
				line := prefix + ui.ProfileName(p.Color, padded) + "  " + ui.Muted(fmt.Sprintf("linked to %d dirs", count))
				if name == c.DefaultProfile {
					line += "  " + ui.Muted("[default]")
				}
				fmt.Fprintln(out, line)
				if showLinks && len(dirs) > 0 {
					sort.Strings(dirs)
					for _, d := range dirs {
						fmt.Fprintln(out, "  "+ui.Path(d))
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&showLinks, "links", false, "show linked directories under each profile")
	return cmd
}
