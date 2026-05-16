package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/commands"
)

const version = "0.0.1-dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "paddock",
		Short:   "Multi-Claude Code account manager",
		Version: version,
	}
	root.AddCommand(
		commands.NewAddCmd(),
		commands.NewListCmd(),
		commands.NewRunCmd(),
		commands.NewWhichCmd(),
		commands.NewLinkCmd(),
		commands.NewUnlinkCmd(),
	)
	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
