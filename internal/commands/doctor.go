// Package commands — doctor.
//
// Spinner deviation: the original plan §6 called for spinners during checks.
// All 11 checks complete in well under 200ms on a healthy system (the slowest
// is the statusline subprocess test, bounded at 2s). A flash spinner is worse
// UX than instant output, so we render results synchronously instead.
package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/profile"
	"github.com/ojuan19/paddock/internal/shell"
	"github.com/ojuan19/paddock/internal/ui"
)

type checkStatus int

const (
	statusOK checkStatus = iota
	statusWarn
	statusError
)

type checkResult struct {
	section string
	status  checkStatus
	line    string
	hint    string
	fixable bool
	fix     func() error
	// fixedLine replaces line in --fix re-render when fix succeeds.
	fixedLine string
}

func NewDoctorCmd() *cobra.Command {
	var doFix bool
	var assumeYes bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose paddock installation; --fix auto-repairs safe issues",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return runDoctor(cmd, doFix, assumeYes) },
	}
	cmd.Flags().BoolVar(&doFix, "fix", false, "Auto-repair fixable issues")
	cmd.Flags().BoolVar(&assumeYes, "yes", false, "Skip confirmations when used with --fix")
	return cmd
}

func runDoctor(cmd *cobra.Command, doFix, assumeYes bool) error {
	out := cmd.OutOrStdout()
	results := collectChecks()
	renderResults(out, results)

	if doFix {
		fixed := applyFixes(cmd, results, assumeYes)
		if fixed > 0 {
			fmt.Fprintln(out, "")
			fmt.Fprintln(out, ui.Muted(fmt.Sprintf("Re-running checks after %d fix(es)...", fixed)))
			fmt.Fprintln(out, "")
			results = collectChecks()
			renderResults(out, results)
		}
	}

	errs, warns := tally(results)
	if errs > 0 {
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, ui.Error(fmt.Sprintf("%d check(s) failed.", errs)))
		os.Exit(1)
	}
	fmt.Fprintln(out, "")
	if warns > 0 {
		fmt.Fprintln(out, ui.Success(fmt.Sprintf("All checks passed (%d warning(s)).", warns)))
	} else {
		fmt.Fprintln(out, ui.Success("All checks passed."))
	}
	return nil
}

func tally(rs []checkResult) (errs, warns int) {
	for _, r := range rs {
		switch r.status {
		case statusError:
			errs++
		case statusWarn:
			warns++
		}
	}
	return
}

func renderResults(out io.Writer, results []checkResult) {
	// Preserve insertion order of sections.
	seen := map[string]bool{}
	order := []string{}
	bySection := map[string][]checkResult{}
	for _, r := range results {
		if !seen[r.section] {
			seen[r.section] = true
			order = append(order, r.section)
		}
		bySection[r.section] = append(bySection[r.section], r)
	}
	for i, section := range order {
		if i > 0 {
			fmt.Fprintln(out, "")
		}
		fmt.Fprintln(out, section)
		for _, r := range bySection[section] {
			fmt.Fprintln(out, "  "+r.line)
			if r.hint != "" {
				fmt.Fprintln(out, "    "+ui.Muted(r.hint))
			}
		}
	}
}

func applyFixes(cmd *cobra.Command, results []checkResult, assumeYes bool) int {
	out := cmd.OutOrStdout()
	in := bufio.NewScanner(cmd.InOrStdin())
	fixed := 0
	for _, r := range results {
		if !r.fixable || r.fix == nil {
			continue
		}
		if r.status == statusOK {
			continue
		}
		// Link-removal fixes prompt unless --yes.
		if strings.HasPrefix(r.fixedLine, "removed") && !assumeYes {
			fmt.Fprintf(out, "Apply fix for: %s [y/N] ", r.line)
			ans := ""
			if in.Scan() {
				ans = strings.ToLower(strings.TrimSpace(in.Text()))
			}
			if ans != "y" && ans != "yes" {
				continue
			}
		}
		if err := r.fix(); err != nil {
			fmt.Fprintln(out, ui.Error("Fix failed: "+err.Error()))
			continue
		}
		fixed++
	}
	return fixed
}

func collectChecks() []checkResult {
	var rs []checkResult
	rs = append(rs, envChecks()...)
	rs = append(rs, configChecks()...)

	cfg, cfgErr := config.Load()
	if cfgErr == nil {
		rs = append(rs, profileChecks(cfg)...)
		rs = append(rs, linkChecks(cfg)...)
	}
	return rs
}

// --- Environment ---

func envChecks() []checkResult {
	var rs []checkResult

	// 1. paddock in PATH
	if path, err := exec.LookPath("paddock"); err == nil {
		rs = append(rs, checkResult{section: "Environment", status: statusOK, line: ui.Success("paddock found at " + path)})
	} else {
		rs = append(rs, checkResult{
			section: "Environment",
			status:  statusError,
			line:    ui.Error("paddock not on PATH"),
			hint:    "Install paddock so its binary is on $PATH (e.g. `go install`).",
		})
	}

	// 2. claude in PATH
	if path, err := exec.LookPath("claude"); err == nil {
		rs = append(rs, checkResult{section: "Environment", status: statusOK, line: ui.Success("claude found at " + path)})
	} else {
		rs = append(rs, checkResult{
			section: "Environment",
			status:  statusError,
			line:    ui.Error("claude not on PATH"),
			hint:    "Install Claude Code: https://docs.claude.com/claude-code",
		})
	}

	// 6. shell hook installed
	rs = append(rs, shellHookCheck())

	return rs
}

func shellHookCheck() checkResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return checkResult{section: "Environment", status: statusError, line: ui.Error("could not locate home dir")}
	}
	sh := filepath.Base(os.Getenv("SHELL"))
	if sh == "" {
		sh = "zsh"
	}
	var rc string
	switch sh {
	case "bash":
		rc = filepath.Join(home, ".bashrc")
	case "fish":
		rc = filepath.Join(home, ".config", "fish", "config.fish")
	default:
		rc = filepath.Join(home, ".zshrc")
	}
	data, err := os.ReadFile(rc)
	if err != nil {
		return checkResult{
			section: "Environment",
			status:  statusWarn,
			line:    ui.Warn("shell hook not installed (" + rcShort(rc) + ")"),
			hint:    "Run: eval \"$(paddock shell-init " + sh + ")\" — then add to " + rcShort(rc),
		}
	}
	if !strings.Contains(string(data), shell.StartMarker) {
		return checkResult{
			section: "Environment",
			status:  statusWarn,
			line:    ui.Warn("shell hook not installed in " + rcShort(rc)),
			hint:    `Add: eval "$(paddock shell-init ` + sh + `)"`,
		}
	}
	return checkResult{section: "Environment", status: statusOK, line: ui.Success("shell hook installed in " + rcShort(rc))}
}

func rcShort(p string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

// --- Config ---

func configChecks() []checkResult {
	var rs []checkResult

	home, _ := os.UserHomeDir()
	paddockHome := filepath.Join(home, ".paddock")
	if info, err := os.Stat(paddockHome); err != nil || !info.IsDir() {
		rs = append(rs, checkResult{
			section: "Config",
			status:  statusError,
			line:    ui.Error("~/.paddock/ does not exist"),
			hint:    "Run `paddock init` to create it.",
		})
		// If the dir doesn't exist, the JSON checks below will also fail; emit them too.
	} else {
		rs = append(rs, checkResult{section: "Config", status: statusOK, line: ui.Success("~/.paddock/ exists")})
	}

	cfg, err := config.Load()
	if err != nil {
		rs = append(rs, checkResult{
			section: "Config",
			status:  statusError,
			line:    ui.Error("config.json invalid"),
			hint:    err.Error(),
		})
	} else {
		rs = append(rs, checkResult{
			section: "Config",
			status:  statusOK,
			line:    ui.Success(fmt.Sprintf("config.json valid (%d profile(s))", len(cfg.Profiles))),
		})
	}

	links, err := config.LoadLinks()
	if err != nil {
		rs = append(rs, checkResult{
			section: "Config",
			status:  statusError,
			line:    ui.Error("links.json invalid"),
			hint:    err.Error(),
		})
	} else {
		rs = append(rs, checkResult{
			section: "Config",
			status:  statusOK,
			line:    ui.Success(fmt.Sprintf("links.json valid (%d link(s))", len(links.Links))),
		})
	}
	return rs
}

// --- Profiles ---

func profileChecks(cfg *config.Config) []checkResult {
	var rs []checkResult

	// 7. default profile exists in profiles map
	if cfg.DefaultProfile != "" {
		if _, ok := cfg.Profiles[cfg.DefaultProfile]; !ok {
			rs = append(rs, checkResult{
				section: "Profiles",
				status:  statusError,
				line:    ui.Error(fmt.Sprintf("default profile %q is not registered", cfg.DefaultProfile)),
				hint:    "Run `paddock add " + cfg.DefaultProfile + "` or edit ~/.paddock/config.json.",
			})
		}
	}

	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		p := cfg.Profiles[name]
		rs = append(rs, profileDirCheck(name, p))
		rs = append(rs, profileSettingsCheck(name, p))
		rs = append(rs, profileLoginCheck(name, p))
		rs = append(rs, profileStatuslineCheck(name, p))
	}
	return rs
}

// 8a. profile dir exists
func profileDirCheck(name string, p config.Profile) checkResult {
	dir, err := profile.Dir(name)
	if err != nil {
		return checkResult{section: "Profiles", status: statusError, line: ui.Error(name + ": invalid name")}
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return checkResult{
			section: "Profiles",
			status:  statusError,
			line:    ui.Error(profileLabel(p.Color, name) + " — profile dir missing"),
			hint:    "Expected: " + rcShort(dir),
			fixable: true,
			fix: func() error {
				_, err := profile.Create(name)
				return err
			},
			fixedLine: "created",
		}
	}
	return checkResult{section: "Profiles", status: statusOK, line: ui.Success(profileLabel(p.Color, name) + " — profile dir ok")}
}

// 8b. settings.json correct (command + color match)
func profileSettingsCheck(name string, p config.Profile) checkResult {
	dir, _ := profile.Dir(name)
	settingsPath := filepath.Join(dir, "settings.json")
	expected := fmt.Sprintf("paddock statusline --profile %s --color %s", name, p.Color)

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return checkResult{
			section: "Profiles",
			status:  statusError,
			line:    ui.Error(profileLabel(p.Color, name) + " — settings.json missing"),
			hint:    "Run `paddock doctor --fix` to regenerate.",
			fixable: true,
			fix:     func() error { return profile.WriteStatuslineSettings(name, p.Color) },
		}
	}
	var parsed struct {
		StatusLine struct {
			Type    string `json:"type"`
			Command string `json:"command"`
		} `json:"statusLine"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return checkResult{
			section: "Profiles",
			status:  statusError,
			line:    ui.Error(profileLabel(p.Color, name) + " — settings.json invalid JSON"),
			hint:    err.Error(),
		}
	}
	if parsed.StatusLine.Command == "" {
		// Pre-existing settings preserved (no statusLine). Not an error.
		return checkResult{
			section: "Profiles",
			status:  statusWarn,
			line:    ui.Warn(profileLabel(p.Color, name) + " — settings.json has no statusLine (preserved on import)"),
		}
	}
	if parsed.StatusLine.Command != expected {
		return checkResult{
			section: "Profiles",
			status:  statusError,
			line:    ui.Error(profileLabel(p.Color, name) + " — settings.json statusLine command mismatch"),
			hint:    "Expected: " + expected,
			fixable: true,
			fix:     func() error { return profile.WriteStatuslineSettings(name, p.Color) },
		}
	}
	return checkResult{section: "Profiles", status: statusOK, line: ui.Success(profileLabel(p.Color, name) + " — settings.json ok")}
}

// 9. .claude.json exists (login proxy, warn only)
func profileLoginCheck(name string, p config.Profile) checkResult {
	dir, _ := profile.Dir(name)
	claudeJSON := filepath.Join(dir, ".claude.json")
	if _, err := os.Stat(claudeJSON); err != nil {
		return checkResult{
			section: "Profiles",
			status:  statusWarn,
			line:    ui.Warn(profileLabel(p.Color, name) + " — not logged in (.claude.json missing)"),
			hint:    "Run `paddock use " + name + " /login` to authenticate.",
		}
	}
	return checkResult{section: "Profiles", status: statusOK, line: ui.Success(profileLabel(p.Color, name) + " — logged in")}
}

// 11. statusline subprocess returns exit 0 with empty JSON stdin (2s timeout)
func profileStatuslineCheck(name string, p config.Profile) checkResult {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	bin, err := exec.LookPath("paddock")
	if err != nil {
		return checkResult{
			section: "Profiles",
			status:  statusError,
			line:    ui.Error(profileLabel(p.Color, name) + " — statusline subprocess check skipped (paddock not on PATH)"),
		}
	}
	cmd := exec.CommandContext(ctx, bin, "statusline", "--profile", name, "--color", p.Color)
	cmd.Stdin = strings.NewReader("{}")
	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return checkResult{
			section: "Profiles",
			status:  statusError,
			line:    ui.Error(profileLabel(p.Color, name) + " — statusline subprocess timed out"),
		}
	}
	if err != nil {
		return checkResult{
			section: "Profiles",
			status:  statusError,
			line:    ui.Error(profileLabel(p.Color, name) + " — statusline subprocess failed: " + err.Error()),
		}
	}
	_ = out
	return checkResult{section: "Profiles", status: statusOK, line: ui.Success(profileLabel(p.Color, name) + " — statusline subprocess ok")}
}

func profileLabel(color, name string) string {
	return ui.ProfileDot(color) + " " + ui.ProfileName(color, name)
}

// --- Links ---

func linkChecks(cfg *config.Config) []checkResult {
	var rs []checkResult
	links, err := config.LoadLinks()
	if err != nil {
		return rs // already reported in config section
	}
	if len(links.Links) == 0 {
		rs = append(rs, checkResult{section: "Links", status: statusOK, line: ui.Muted("(no links configured)")})
		return rs
	}
	dirs := make([]string, 0, len(links.Links))
	for d := range links.Links {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		profName := links.Links[dir]
		dirCopy := dir
		// target dir exists?
		info, err := os.Stat(dir)
		dirOK := err == nil && info.IsDir()
		_, profOK := cfg.Profiles[profName]
		switch {
		case !dirOK:
			rs = append(rs, checkResult{
				section: "Links",
				status:  statusError,
				line:    ui.Error("broken link " + dir + " → " + profName),
				hint:    "Target dir no longer exists.",
				fixable: true,
				fix: func() error {
					l, err := config.LoadLinks()
					if err != nil {
						return err
					}
					delete(l.Links, dirCopy)
					return l.Save()
				},
				fixedLine: "removed",
			})
		case !profOK:
			rs = append(rs, checkResult{
				section: "Links",
				status:  statusError,
				line:    ui.Error("link " + dir + " → " + profName + " references unknown profile"),
				hint:    "Run `paddock unlink " + dir + "` or recreate the profile.",
			})
		default:
			color := cfg.Profiles[profName].Color
			rs = append(rs, checkResult{
				section: "Links",
				status:  statusOK,
				line:    ui.Success(dir + " → " + profileLabel(color, profName)),
			})
		}
	}
	return rs
}
