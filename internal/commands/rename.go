package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/profile"
	"github.com/ojuan19/paddock/internal/ui"
)

func NewRenameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a profile (config, directory, links, .paddock files)",
		Long: "Rename a profile. Updates config.json, renames the on-disk profile " +
			"directory, rewrites the statusline command in the profile's settings.json, " +
			"updates links.json, and rewrites any .paddock files at known link paths " +
			"that still point to the old name. Active claude sessions using the old " +
			"name should be restarted.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRename(cmd, args[0], args[1])
		},
	}
	return cmd
}

func runRename(cmd *cobra.Command, oldName, newName string) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	if oldName == newName {
		return errors.New("old and new names are the same")
	}
	if err := profile.ValidateName(newName); err != nil {
		return fmt.Errorf("invalid new name %q: %w", newName, err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	oldProf, ok := cfg.Profiles[oldName]
	if !ok {
		msg := ui.Error(fmt.Sprintf("No profile named %q", oldName))
		if s := ui.SuggestProfile(oldName, profileNames(cfg)); s != "" {
			msg += "\n" + s
		}
		return errors.New(msg)
	}
	if _, exists := cfg.Profiles[newName]; exists {
		return fmt.Errorf("profile %q already exists", newName)
	}

	links, err := config.LoadLinks()
	if err != nil {
		return fmt.Errorf("loading links: %w", err)
	}

	oldDir, err := profile.Dir(oldName)
	if err != nil {
		return err
	}
	newDir, err := profile.Dir(newName)
	if err != nil {
		return err
	}
	if info, err := os.Stat(oldDir); err != nil || !info.IsDir() {
		return fmt.Errorf("profile %q is registered but its directory is missing: %s", oldName, oldDir)
	}
	if _, err := os.Stat(newDir); err == nil {
		return fmt.Errorf("target directory already exists: %s", newDir)
	}

	// Phase B — mutations with rollback stack. Order: dir rename → settings →
	// config → links → .paddock files. The .paddock step is best-effort and
	// intentionally has no rollback; partial failure there leaves a clear
	// resolver error but does not corrupt config.
	var rollbacks []func()
	rollback := func() {
		for i := len(rollbacks) - 1; i >= 0; i-- {
			rollbacks[i]()
		}
	}

	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("renaming profile directory: %w", err)
	}
	rollbacks = append(rollbacks, func() { _ = os.Rename(newDir, oldDir) })

	// Settings.json lives inside the renamed dir; reverting the dir rename
	// restores the original settings.json, so no separate rollback needed.
	if err := profile.UpdateStatuslineCommand(newName, oldProf.Color); err != nil {
		rollback()
		return fmt.Errorf("updating statusline settings: %w", err)
	}

	if err := cfg.Rename(oldName, newName); err != nil {
		rollback()
		return fmt.Errorf("updating config: %w", err)
	}
	if err := cfg.Save(); err != nil {
		// In-memory rename will be undone by the rollback closure below; but
		// since we haven't pushed it yet, do it inline.
		_ = cfg.Rename(newName, oldName)
		rollback()
		return fmt.Errorf("saving config: %w", err)
	}
	rollbacks = append(rollbacks, func() {
		if err := cfg.Rename(newName, oldName); err == nil {
			_ = cfg.Save()
		}
	})

	affectedPaths := links.RenameProfile(oldName, newName)
	if len(affectedPaths) > 0 {
		if err := links.Save(); err != nil {
			rollback()
			return fmt.Errorf("saving links: %w", err)
		}
	}

	// Best-effort .paddock file rewrite. Only rewrite a file if it exists and
	// its trimmed contents exactly match oldName — defensive against a user
	// who hand-edited a .paddock to point somewhere else.
	rewritten, skipped := rewriteDotPaddockFiles(affectedPaths, oldName, newName)

	dot := ui.ProfileDot(oldProf.Color)
	dotSp := ""
	if dot != "" {
		dotSp = dot + " "
	}
	fmt.Fprintln(out, ui.Success("Renamed ")+ui.Muted(oldName)+" → "+dotSp+ui.ProfileName(oldProf.Color, newName))
	if len(affectedPaths) > 0 {
		fmt.Fprintf(out, "  updated %d link(s) in links.json\n", len(affectedPaths))
	}
	if rewritten > 0 {
		fmt.Fprintf(out, "  rewrote %d .paddock file(s)\n", rewritten)
	}
	for _, s := range skipped {
		fmt.Fprintln(errOut, ui.Warn(s))
	}
	fmt.Fprintln(out, ui.Muted("  note: any active claude session using \""+oldName+"\" should be restarted."))
	return nil
}

// rewriteDotPaddockFiles iterates the given link paths, looks for a .paddock
// file in each, and rewrites it to newName only if its trimmed contents are
// exactly oldName. Returns (rewritten count, warning messages for files that
// could not be rewritten — e.g. read errors).
func rewriteDotPaddockFiles(paths []string, oldName, newName string) (int, []string) {
	var rewritten int
	var warnings []string
	for _, p := range paths {
		dot := filepath.Join(p, ".paddock")
		data, err := os.ReadFile(dot)
		if err != nil {
			if !os.IsNotExist(err) {
				warnings = append(warnings, fmt.Sprintf("could not read %s: %v", dot, err))
			}
			continue
		}
		if strings.TrimSpace(string(data)) != oldName {
			// Hand-edited to point elsewhere; leave alone.
			continue
		}
		if err := config.WriteFileAtomic(dot, []byte(newName+"\n")); err != nil {
			warnings = append(warnings, fmt.Sprintf("could not rewrite %s: %v", dot, err))
			continue
		}
		rewritten++
	}
	return rewritten, warnings
}
