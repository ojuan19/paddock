package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/shell"
	"github.com/ojuan19/paddock/internal/ui"
)

// isStdinTTY is var-bound so tests can override it.
var isStdinTTY = func() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) != 0
}

func NewLinkCmd() *cobra.Command {
	var assumeYes bool
	cmd := &cobra.Command{
		Use:   "link [name]",
		Short: "Link the current directory to a profile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			if len(cfg.Profiles) == 0 {
				return errors.New("no profiles yet. Run 'paddock add <name>' first")
			}
			var name string
			if len(args) == 1 {
				name = args[0]
			} else {
				name, err = pickProfile(cmd, cfg)
				if err != nil {
					return err
				}
				if name == "" {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}
			if err := runLink(cmd, cfg, name); err != nil {
				return err
			}
			// Shell-hook install prompt — non-fatal on error.
			maybePromptShellInstall(cmd, assumeYes)
			return nil
		},
	}
	cmd.Flags().BoolVar(&assumeYes, "yes", false, "Skip the shell-integration install prompt")
	return cmd
}

// canonicalPwd intentionally skips EvalSymlinks to mirror the resolver.
func canonicalPwd() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	abs, err := filepath.Abs(wd)
	if err != nil {
		return "", fmt.Errorf("canonicalizing working directory: %w", err)
	}
	return abs, nil
}

func runLink(cmd *cobra.Command, cfg *config.Config, name string) error {
	if _, ok := cfg.Profiles[name]; !ok {
		msg := ui.Error(fmt.Sprintf("No profile named %q", name))
		if s := ui.SuggestProfile(name, profileNames(cfg)); s != "" {
			msg += "\n" + s
		}
		return errors.New(msg)
	}
	pwd, err := canonicalPwd()
	if err != nil {
		return err
	}
	links, err := config.LoadLinks()
	if err != nil {
		return fmt.Errorf("loading links: %w", err)
	}
	if links.Links == nil {
		links.Links = map[string]string{}
	}
	out := cmd.OutOrStdout()
	p := cfg.Profiles[name]
	dot := ui.ProfileDot(p.Color)
	dotSp := ""
	if dot != "" {
		dotSp = dot + " "
	}
	existing, has := links.Links[pwd]
	switch {
	case !has:
		links.Links[pwd] = name
		if err := links.Save(); err != nil {
			return fmt.Errorf("saving links: %w", err)
		}
		fmt.Fprintln(out, ui.Success("Linked ")+dotSp+ui.ProfileName(p.Color, name))
		fmt.Fprintln(out, "  "+ui.Path(pwd))
	case existing == name:
		fmt.Fprintln(out, "Already linked: "+dotSp+ui.ProfileName(p.Color, name))
		fmt.Fprintln(out, "  "+ui.Path(pwd))
	default:
		oldP := cfg.Profiles[existing]
		oldDot := ui.ProfileDot(oldP.Color)
		oldDotSp := ""
		if oldDot != "" {
			oldDotSp = oldDot + " "
		}
		links.Links[pwd] = name
		if err := links.Save(); err != nil {
			return fmt.Errorf("saving links: %w", err)
		}
		fmt.Fprintln(out, ui.Success("Re-linked ")+dotSp+ui.ProfileName(p.Color, name))
		fmt.Fprintln(out, "  was: "+oldDotSp+ui.ProfileName(oldP.Color, existing))
		fmt.Fprintln(out, "  "+ui.Path(pwd))
	}
	return nil
}

// maybePromptShellInstall checks for a missing shell hook and offers to install it.
// All branches are best-effort — link has already succeeded by the time we get here.
func maybePromptShellInstall(cmd *cobra.Command, assumeYes bool) {
	out := cmd.OutOrStdout()

	name, rc, supported := shell.DetectShell()
	if !supported {
		return
	}
	installed, err := shell.IsInstalled(rc)
	if err != nil || installed {
		return
	}

	rcShort := rcShortPath(rc)

	if !assumeYes && !isStdinTTY() {
		return
	}

	accept := assumeYes
	if !assumeYes {
		fmt.Fprintf(out, "Shell auto-switch not installed. Add to %s? [Y/n] ", rcShort)
		sc := bufio.NewScanner(cmd.InOrStdin())
		if sc.Scan() {
			ans := strings.ToLower(strings.TrimSpace(sc.Text()))
			accept = ans == "" || ans == "y" || ans == "yes"
		}
	}

	if !accept {
		fmt.Fprintf(out, "Skipped. To install later: paddock shell-init %s >> %s\n", name, rcShort)
		return
	}

	// Note the bash_profile state BEFORE we write, so we can emit the macOS hint accurately.
	hadBashrcBefore := false
	if name == "bash" {
		if _, statErr := os.Stat(rc); statErr == nil {
			hadBashrcBefore = true
		}
	}

	if err := shell.Install(name, rc); err != nil {
		fmt.Fprintln(out, ui.Warn("Failed to install shell hook: "+err.Error()))
		fmt.Fprintf(out, "To install later: paddock shell-init %s >> %s\n", name, rcShort)
		return
	}
	fmt.Fprintf(out, "Added paddock hook to %s. Run `source %s` or open a new shell to activate.\n", rcShort, rcShort)

	if name == "bash" && !hadBashrcBefore {
		home, err := os.UserHomeDir()
		if err == nil {
			if _, err := os.Stat(filepath.Join(home, ".bash_profile")); err == nil {
				fmt.Fprintf(out, "Note: your shell may load ~/.bash_profile; ensure it sources %s.\n", rcShort)
			}
		}
	}
}

func rcShortPath(p string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

func pickProfile(cmd *cobra.Command, cfg *config.Config) (string, error) {
	// TTY check: refuse interactive picker when stdin is piped/redirected.
	if !isStdinTTY() {
		msg := ui.Error("Interactive picker requires a TTY") +
			"\n" + ui.Muted("  Pass a profile name: paddock link <name>")
		return "", errors.New(msg)
	}
	names := profileNames(cfg)
	out := cmd.OutOrStdout()
	for i, n := range names {
		p := cfg.Profiles[n]
		dot := ui.ProfileDot(p.Color)
		dotSp := ""
		if dot != "" {
			dotSp = dot + " "
		}
		if n == cfg.DefaultProfile {
			fmt.Fprintf(out, "%d) %s%s %s\n", i+1, dotSp, ui.ProfileName(p.Color, n), ui.Muted("[default]"))
		} else {
			fmt.Fprintf(out, "%d) %s%s\n", i+1, dotSp, ui.ProfileName(p.Color, n))
		}
	}
	fmt.Fprintf(out, "Pick a profile (1-%d, q to abort): ", len(names))
	scanner := bufio.NewScanner(cmd.InOrStdin())
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())
	if input == "" || input == "q" || input == "Q" {
		return "", nil
	}
	n, err := strconv.Atoi(input)
	if err != nil || n < 1 || n > len(names) {
		return "", errors.New("invalid selection")
	}
	return names[n-1], nil
}
