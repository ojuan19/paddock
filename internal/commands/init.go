package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/profile"
	"github.com/ojuan19/paddock/internal/ui"
)

func NewInitCmd() *cobra.Command {
	var assumeYes bool
	var name string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "First-run setup; offers to import ~/.claude/ as a profile",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return runInit(cmd, assumeYes, name) },
	}
	cmd.Flags().BoolVar(&assumeYes, "yes", false, "Skip the import prompt and accept")
	cmd.Flags().StringVar(&name, "name", "personal", "Profile name to use for the imported config")
	return cmd
}

func runInit(cmd *cobra.Command, assumeYes bool, name string) error {
	out := cmd.OutOrStdout()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Profiles) > 0 {
		fmt.Fprintf(out, "Already initialized (%d profiles). Nothing to do.\n", len(cfg.Profiles))
		return nil
	}

	// Ensure ~/.paddock/ exists even if we bail early (so subsequent commands have a home).
	if _, err := config.Path(); err != nil {
		return fmt.Errorf("preparing paddock dir: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("locating home dir: %w", err)
	}
	claudeHome := filepath.Join(home, ".claude")
	info, statErr := os.Stat(claudeHome)
	if statErr != nil || !info.IsDir() {
		fmt.Fprintln(out, ui.Muted("No existing ~/.claude/ found."))
		fmt.Fprintln(out, "Run "+ui.Bold("paddock add <name>")+" to create your first profile.")
		printShellInitHint(out)
		return nil
	}

	if err := profile.ValidateName(name); err != nil {
		return fmt.Errorf("invalid profile name %q: %w", name, err)
	}

	accept := assumeYes
	if !assumeYes {
		fmt.Fprintf(out, "Import ~/.claude/ as profile %q? [Y/n] ", name)
		sc := bufio.NewScanner(cmd.InOrStdin())
		if sc.Scan() {
			answer := strings.ToLower(strings.TrimSpace(sc.Text()))
			accept = answer == "" || answer == "y" || answer == "yes"
		}
	}

	if !accept {
		fmt.Fprintln(out, ui.Muted("Skipped import."))
		printShellInitHint(out)
		return nil
	}

	if _, err := profile.Create(name); err != nil {
		return fmt.Errorf("creating profile dir: %w", err)
	}
	sum, err := profile.ImportFrom(claudeHome, name)
	if err != nil {
		return fmt.Errorf("importing ~/.claude: %w", err)
	}

	// Copy ~/.claude.json (lives at home root, not inside ~/.claude/)
	claudeJSONSrc := filepath.Join(home, ".claude.json")
	if info, err := os.Lstat(claudeJSONSrc); err == nil && info.Mode().IsRegular() {
		data, readErr := os.ReadFile(claudeJSONSrc)
		if readErr == nil {
			dstPath := filepath.Join(mustProfileDir(name), ".claude.json")
			if writeErr := config.WriteFileAtomic(dstPath, data); writeErr == nil {
				// Match the 0o600 perm of the source intent — auth token inside.
				_ = os.Chmod(dstPath, 0o600)
				sum.Files[".claude.json"]++
			}
		}
	}

	color := cfg.AutoAssignColor()
	cfg.Profiles[name] = config.Profile{Color: color, CreatedAt: time.Now().UTC()}
	cfg.DefaultProfile = name
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	srcSettingsPath := filepath.Join(claudeHome, "settings.json")
	profileSettingsPath := filepath.Join(mustProfileDir(name), "settings.json")
	if profile.HasExistingStatusLine(profileSettingsPath) || profile.HasExistingStatusLine(srcSettingsPath) {
		fmt.Fprintln(out, ui.Warn("Existing statusLine in settings.json preserved."))
		fmt.Fprintln(out, ui.Muted("  Paddock's colored statusline is disabled for this profile."))
	} else {
		if err := profile.WriteStatuslineSettings(name, color); err != nil {
			return fmt.Errorf("writing statusline settings: %w", err)
		}
	}

	fmt.Fprintf(out, "%s %s %s  %s\n",
		ui.Success("Imported as"),
		ui.ProfileDot(color),
		ui.ProfileName(color, name),
		ui.Muted("(color: "+color+", default)"),
	)
	keys := make([]string, 0, len(sum.Files))
	for k := range sum.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s (%d)", k, sum.Files[k]))
	}
	if len(parts) > 0 {
		fmt.Fprintln(out, ui.Muted("  Files: "+strings.Join(parts, ", ")))
	}
	printShellInitHint(out)
	return nil
}

func printShellInitHint(out io.Writer) {
	sh := filepath.Base(os.Getenv("SHELL"))
	if sh == "" {
		sh = "zsh"
	}
	switch sh {
	case "bash":
		fmt.Fprintln(out, "\nNext: add to ~/.bashrc:")
		fmt.Fprintln(out, "  "+ui.Bold(`eval "$(paddock shell-init bash)"`))
	case "fish":
		fmt.Fprintln(out, "\nNext: add to ~/.config/fish/config.fish:")
		fmt.Fprintln(out, "  "+ui.Bold(`paddock shell-init fish | source`))
	default:
		fmt.Fprintln(out, "\nNext: add to ~/.zshrc:")
		fmt.Fprintln(out, "  "+ui.Bold(`eval "$(paddock shell-init zsh)"`))
	}
}

func mustProfileDir(name string) string {
	p, _ := profile.Dir(name)
	return p
}
