package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/resolver"
	"github.com/ojuan19/paddock/internal/ui"
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

	// --quiet bypasses ALL UI styling. Hot path for the cd-hook in Round 6.
	if quiet {
		if err != nil {
			return err
		}
		if result.Rule == resolver.RuleNone {
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), result.Profile)
		return nil
	}

	if err != nil {
		var nice *resolver.NotInConfigError
		if errors.As(err, &nice) {
			msg := ui.Error(nice.Error())
			if s := ui.SuggestProfile(nice.Name, profileNames(cfg)); s != "" {
				msg += "\n" + s
			}
			return errors.New(msg)
		}
		return err
	}

	out := cmd.OutOrStdout()
	if result.Rule == resolver.RuleNone {
		fmt.Fprintln(out, ui.Muted("No profile applies here."))
		return nil
	}

	p := cfg.Profiles[result.Profile]
	dot := ui.ProfileDot(p.Color)
	dotSp := ""
	if dot != "" {
		dotSp = dot + " "
	}
	fmt.Fprintln(out, dotSp+ui.ProfileName(p.Color, result.Profile))
	fmt.Fprintln(out, "  "+ui.Muted("source: "+sourceLabel(result)))
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
