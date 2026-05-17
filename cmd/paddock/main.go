package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/commands"
)

var version = "dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "paddock",
		Short:   "Multi-Claude Code account manager",
		Version: version,
	}
	root.AddCommand(
		commands.NewInitCmd(),
		commands.NewAddCmd(),
		commands.NewListCmd(),
		commands.NewRunCmd(),
		commands.NewUseCmd(),
		commands.NewWhichCmd(),
		commands.NewLinkCmd(),
		commands.NewUnlinkCmd(),
		commands.NewStatuslineCmd(),
		commands.NewShellInitCmd(),
		commands.NewDoctorCmd(),
	)
	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
