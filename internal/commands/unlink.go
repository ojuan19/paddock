package commands

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/ui"
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
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			links, err := config.LoadLinks()
			if err != nil {
				return fmt.Errorf("loading links: %w", err)
			}
			existing, has := links.Links[pwd]
			if !has {
				msg := ui.Error("No link for this directory") + "\n  " + ui.Path(pwd)
				return errors.New(msg)
			}
			delete(links.Links, pwd)
			if err := links.Save(); err != nil {
				return fmt.Errorf("saving links: %w", err)
			}
			out := cmd.OutOrStdout()
			p := cfg.Profiles[existing]
			dot := ui.ProfileDot(p.Color)
			dotSp := ""
			if dot != "" {
				dotSp = dot + " "
			}
			fmt.Fprintln(out, ui.Success("Unlinked ")+dotSp+ui.ProfileName(p.Color, existing))
			fmt.Fprintln(out, "  "+ui.Path(pwd))
			return nil
		},
	}
	return cmd
}
