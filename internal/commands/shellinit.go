package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/shell"
)

func NewShellInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "shell-init [bash|zsh|fish|powershell]",
		Short:     "Print shell integration script for eval'ing into your shell rc",
		Long:      "Print a shell snippet to be eval'd into your shell rc file.\nPowerShell support is experimental.",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			script, err := shell.Script(args[0])
			if err != nil {
				return fmt.Errorf("getting shell script: %w", err)
			}
			fmt.Fprint(cmd.OutOrStdout(), script)
			return nil
		},
	}
	return cmd
}
