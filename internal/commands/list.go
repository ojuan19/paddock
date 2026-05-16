package commands

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
)

func NewListCmd() *cobra.Command {
	return &cobra.Command{
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
				fmt.Fprintln(out, "No profiles yet. Run 'paddock add <name>'.")
				return nil
			}

			names := make([]string, 0, len(c.Profiles))
			for n := range c.Profiles {
				names = append(names, n)
			}
			sort.Strings(names)

			for _, name := range names {
				count := 0
				for _, p := range links.Links {
					if p == name {
						count++
					}
				}
				line := fmt.Sprintf("%s  linked to %d dirs", name, count)
				if name == c.DefaultProfile {
					line += "\t[default]"
				}
				fmt.Fprintln(out, line)
			}
			return nil
		},
	}
}
