package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/resolver"
)

func NewWhichCmd() *cobra.Command {
	var quiet bool
	cmd := &cobra.Command{
		Use:   "which",
		Short: "Show which profile applies here and why",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhich(cmd, quiet)
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Print only the profile name")
	return cmd
}

func runWhich(cmd *cobra.Command, quiet bool) error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	links, err := config.LoadLinks()
	if err != nil {
		return fmt.Errorf("loading links: %w", err)
	}

	result, err := resolver.Resolve(pwd, cfg, links)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if quiet {
		if result.Rule == resolver.RuleNone {
			return nil
		}
		fmt.Fprintln(out, result.Profile)
		return nil
	}

	if result.Rule == resolver.RuleNone {
		fmt.Fprintln(out, "No profile applies here.")
		return nil
	}

	fmt.Fprintf(out, "Profile: %s\n", result.Profile)
	fmt.Fprintf(out, "Source:  %s\n", sourceLabel(result))
	return nil
}

func sourceLabel(r resolver.Result) string {
	switch r.Rule {
	case resolver.RuleEnvVar:
		return "$PADDOCK_PROFILE"
	case resolver.RuleDotPaddockFile:
		return fmt.Sprintf(".paddock file at %s", r.Source)
	case resolver.RuleLinksJSON:
		return fmt.Sprintf("links.json entry for %s", r.Source)
	case resolver.RuleDefaultProfile:
		return "default_profile in config.json"
	default:
		return ""
	}
}
