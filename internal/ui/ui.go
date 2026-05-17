package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var colorEnabled bool

var profileColors = map[string]lipgloss.Color{
	"blue":   "#5B9BD5",
	"green":  "#52C77E",
	"amber":  "#E0A95C",
	"red":    "#E06C6C",
	"purple": "#9C7BD0",
	"teal":   "#4FB8B0",
	"pink":   "#E59FC8",
	"coral":  "#E89070",
	"gray":   "#888888",
}

const grayHex lipgloss.Color = "#888888"

func init() {
	// Precedence: NO_COLOR present → off; FORCE_COLOR non-empty → on; else TTY check.
	colorEnabled = detectColor()
	if colorEnabled {
		lipgloss.SetColorProfile(termenv.TrueColor)
	} else {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}

func detectColor() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	if v, ok := os.LookupEnv("FORCE_COLOR"); ok && v != "" {
		return true
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func ColorEnabled() bool { return colorEnabled }

// ForceColor enables colored output for the rest of this process, overriding
// TTY detection and NO_COLOR. Used by `paddock statusline` because its stdout
// is always a pipe (Claude Code reads it) but Claude Code does render ANSI.
func ForceColor() {
	colorEnabled = true
	lipgloss.SetColorProfile(termenv.TrueColor)
}

func ResolveColor(name string) lipgloss.Color {
	if c, ok := profileColors[name]; ok {
		return c
	}
	return grayHex
}

// Non-TTY mode strips unicode glyphs entirely so output stays grep-friendly.
func ProfileDot(color string) string {
	if !colorEnabled {
		return ""
	}
	return lipgloss.NewStyle().Foreground(ResolveColor(color)).Render("●")
}

func ProfileName(color, name string) string {
	if !colorEnabled {
		return name
	}
	return lipgloss.NewStyle().Foreground(ResolveColor(color)).Render(name)
}

func Success(s string) string {
	if !colorEnabled {
		return s
	}
	return lipgloss.NewStyle().Foreground(ResolveColor("green")).Render("✓ " + s)
}

func Error(s string) string {
	if !colorEnabled {
		return s
	}
	return lipgloss.NewStyle().Foreground(ResolveColor("red")).Render("✗ " + s)
}

func Warn(s string) string {
	if !colorEnabled {
		return s
	}
	return lipgloss.NewStyle().Foreground(ResolveColor("amber")).Render("⚠ " + s)
}

func Muted(s string) string {
	if !colorEnabled {
		return s
	}
	return lipgloss.NewStyle().Foreground(grayHex).Faint(true).Render(s)
}

func Bold(s string) string {
	if !colorEnabled {
		return s
	}
	return lipgloss.NewStyle().Bold(true).Render(s)
}

func Path(s string) string { return Muted(s) }

func SuggestProfile(badName string, all []string) string {
	if len(all) == 0 {
		return ""
	}
	threshold := len(badName) / 3
	if threshold < 2 {
		threshold = 2
	}
	bad := strings.ToLower(badName)
	bestDist := -1
	var bestNames []string
	for _, n := range all {
		d := levenshtein(bad, strings.ToLower(n))
		if d > threshold {
			continue
		}
		if bestDist == -1 || d < bestDist {
			bestDist = d
			bestNames = []string{n}
		} else if d == bestDist {
			bestNames = append(bestNames, n)
		}
	}
	if len(bestNames) == 0 {
		return ""
	}
	sort.Strings(bestNames)
	return Muted(fmt.Sprintf("  Did you mean '%s'? Run 'paddock list' to see all profiles.", bestNames[0]))
}

func levenshtein(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	la, lb := len(ar), len(br)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			m := del
			if ins < m {
				m = ins
			}
			if sub < m {
				m = sub
			}
			curr[j] = m
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}
