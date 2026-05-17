package commands

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/ui"
)

func NewStatuslineCmd() *cobra.Command {
	var profile, color string
	cmd := &cobra.Command{
		Use:    "statusline",
		Short:  "(internal) Statusline command for Claude Code",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ui.ForceColor()
			// Drain stdin first — Claude Code writes session JSON and we discard it.
			// Not draining can leave Claude with a SIGPIPE on close.
			_, _ = io.Copy(io.Discard, cmd.InOrStdin())
			name := lipgloss.NewStyle().Foreground(ui.ResolveColor(color)).Render(profile)
			fmt.Fprint(cmd.OutOrStdout(), "["+name+"]")
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "", "Profile name to display")
	cmd.Flags().StringVar(&color, "color", "", "Color of the profile")
	_ = cmd.MarkFlagRequired("profile")
	_ = cmd.MarkFlagRequired("color")
	return cmd
}
