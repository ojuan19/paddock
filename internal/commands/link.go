package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
)

func NewLinkCmd() *cobra.Command {
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
			return runLink(cmd, cfg, name)
		},
	}
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
		return fmt.Errorf("%w: %q", config.ErrProfileNotFound, name)
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
	existing, has := links.Links[pwd]
	switch {
	case !has:
		links.Links[pwd] = name
		if err := links.Save(); err != nil {
			return fmt.Errorf("saving links: %w", err)
		}
		fmt.Fprintf(out, "Linked %s → %s\n", pwd, name)
	case existing == name:
		fmt.Fprintf(out, "Already linked: %s → %s\n", pwd, name)
	default:
		links.Links[pwd] = name
		if err := links.Save(); err != nil {
			return fmt.Errorf("saving links: %w", err)
		}
		fmt.Fprintf(out, "Re-linked %s: %s → %s\n", pwd, existing, name)
	}
	return nil
}

func pickProfile(cmd *cobra.Command, cfg *config.Config) (string, error) {
	// TTY check: refuse interactive picker when stdin is piped/redirected.
	fi, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		return "", errors.New("interactive picker requires a TTY; pass a profile name explicitly: paddock link <name>")
	}
	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	out := cmd.OutOrStdout()
	for i, n := range names {
		if n == cfg.DefaultProfile {
			fmt.Fprintf(out, "%d) %s [default]\n", i+1, n)
		} else {
			fmt.Fprintf(out, "%d) %s\n", i+1, n)
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
