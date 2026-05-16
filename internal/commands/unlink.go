package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
)

func NewUnlinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlink",
		Short: "Remove the link binding the current directory to a profile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pwd, err := canonicalPwd()
			if err != nil {
				return err
			}
			links, err := config.LoadLinks()
			if err != nil {
				return fmt.Errorf("loading links: %w", err)
			}
			existing, has := links.Links[pwd]
			if !has {
				return fmt.Errorf("no link for %s", pwd)
			}
			delete(links.Links, pwd)
			if err := links.Save(); err != nil {
				return fmt.Errorf("saving links: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Unlinked %s (was %s)\n", pwd, existing)
			return nil
		},
	}
	return cmd
}
