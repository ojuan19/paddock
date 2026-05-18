package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/profile"
	"github.com/ojuan19/paddock/internal/ui"
)

// Note: shell-hook install prompt lives in link.go, not here.
// The hook isn't load-bearing until a directory is linked.

func NewAddCmd() *cobra.Command {
	var colorFlag string
	var defaultFlag bool
	var fromDir string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new profile (optionally importing from an existing CLAUDE_CONFIG_DIR)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(cmd, args[0], colorFlag, defaultFlag, fromDir)
		},
	}

	cmd.Flags().StringVar(&colorFlag, "color", "", "color label for the profile")
	cmd.Flags().BoolVar(&defaultFlag, "default", false, "set as the default profile")
	cmd.Flags().StringVar(&fromDir, "from", "", "Import an existing CLAUDE_CONFIG_DIR setup (.claude.json, settings.json, agents/, commands/, skills/, plugins/)")
	return cmd
}

func runAdd(cmd *cobra.Command, name, colorFlag string, defaultFlag bool, fromDir string) error {
	if err := profile.ValidateName(name); err != nil {
		return fmt.Errorf("%w: %q (allowed: letters, digits, '-', '_')", err, name)
	}

	c, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if _, ok := c.Profiles[name]; ok {
		return errors.New(ui.Error(fmt.Sprintf("Profile %q already exists", name)))
	}

	var color string
	if colorFlag != "" {
		if !slices.Contains(config.AllowedColors, colorFlag) {
			msg := ui.Error(fmt.Sprintf("Invalid color %q", colorFlag)) +
				"\n" + ui.Muted("  Allowed: "+strings.Join(config.AllowedColors, ", "))
			return errors.New(msg)
		}
		color = colorFlag
	} else {
		color = c.AutoAssignColor()
	}

	// Validate --from BEFORE creating anything
	var fromAbs string
	if fromDir != "" {
		fromAbs, err = filepath.Abs(fromDir)
		if err != nil {
			return fmt.Errorf("resolving --from path: %w", err)
		}
		info, statErr := os.Stat(fromAbs)
		if statErr != nil || !info.IsDir() {
			return fmt.Errorf("--from %q: not a directory", fromDir)
		}
		destDir, dirErr := profile.Dir(name)
		if dirErr != nil {
			return dirErr
		}
		destAbs, _ := filepath.Abs(destDir)
		if filepath.Clean(fromAbs) == filepath.Clean(destAbs) {
			return errors.New("--from cannot be the profile's own directory")
		}
		hasClaudeJSON := fileExists(filepath.Join(fromAbs, ".claude.json"))
		hasSettings := fileExists(filepath.Join(fromAbs, "settings.json"))
		if !hasClaudeJSON && !hasSettings {
			return fmt.Errorf("--from %q: not a Claude config dir (no .claude.json or settings.json found)", fromDir)
		}
	}

	if _, err := profile.Create(name); err != nil {
		return fmt.Errorf("creating profile dir: %w", err)
	}

	var sum profile.ImportSummary
	if fromAbs != "" {
		sum, err = profile.ImportFrom(fromAbs, name)
		if err != nil {
			return fmt.Errorf("importing from %s: %w", fromDir, err)
		}
	}

	c.Profiles[name] = config.Profile{
		Color:     color,
		CreatedAt: time.Now().UTC(),
	}

	isDefault := defaultFlag || len(c.Profiles) == 1
	if isDefault {
		c.DefaultProfile = name
	}

	if err := c.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	out := cmd.OutOrStdout()

	profileSettingsPath := filepath.Join(mustProfileDir(name), "settings.json")
	skipStatusline := false
	if fromAbs != "" {
		srcSettings := filepath.Join(fromAbs, "settings.json")
		if profile.HasExistingStatusLine(profileSettingsPath) || profile.HasExistingStatusLine(srcSettings) {
			skipStatusline = true
		}
	}
	if !skipStatusline {
		if err := profile.WriteStatuslineSettings(name, color); err != nil {
			return fmt.Errorf("writing statusline settings: %w", err)
		}
	} else {
		fmt.Fprintln(out, ui.Warn("Existing statusLine in settings.json preserved."))
		fmt.Fprintln(out, ui.Muted("  Paddock's colored statusline is disabled for this profile."))
	}

	suffix := color
	if isDefault {
		suffix = color + ", default"
	}
	line := ui.Success("Created profile ") + ui.ProfileDot(color) + " " + ui.ProfileName(color, name) +
		"  " + ui.Muted("("+suffix+")")
	fmt.Fprintln(out, line)

	if fromAbs != "" && len(sum.Files) > 0 {
		keys := make([]string, 0, len(sum.Files))
		for k := range sum.Files {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s (%d)", k, sum.Files[k]))
		}
		fmt.Fprintln(out, ui.Muted("  Files: "+strings.Join(parts, ", ")))
	}

	return nil
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
