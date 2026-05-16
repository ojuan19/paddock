package resolver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ojuan19/paddock/internal/config"
	"github.com/ojuan19/paddock/internal/profile"
)

type Rule int

const (
	RuleNone Rule = iota
	RuleEnvVar
	RuleDotPaddockFile
	RuleLinksJSON
	RuleDefaultProfile
)

type Result struct {
	Profile string
	Rule    Rule
	Source  string
}

var (
	ErrInvalidDotPaddock  = errors.New(".paddock file is invalid")
	ErrProfileNotInConfig = errors.New("profile referenced but not in config")
)

type NotInConfigError struct {
	Name   string
	Source string
}

func (e *NotInConfigError) Error() string {
	return fmt.Sprintf("%s: profile not in config: %q", e.Source, e.Name)
}

func (e *NotInConfigError) Is(target error) bool {
	return target == ErrProfileNotInConfig
}

func Resolve(pwd string, cfg *config.Config, links *config.Links) (Result, error) {
	// Rule 1: $PADDOCK_PROFILE
	if val, ok := os.LookupEnv("PADDOCK_PROFILE"); ok && val != "" {
		if _, exists := cfg.Profiles[val]; !exists {
			return Result{}, &NotInConfigError{Name: val, Source: "$PADDOCK_PROFILE"}
		}
		return Result{Profile: val, Rule: RuleEnvVar}, nil
	}

	// Rule 2: .paddock file walk — start at pwd, walk to parent, stop at $HOME or filesystem root
	// (avoids leaking past the user's home into shared system dirs).
	stopAt := homeStop(pwd)
	cur := pwd
	for {
		dotPath := filepath.Join(cur, ".paddock")
		if data, err := os.ReadFile(dotPath); err == nil {
			name := strings.TrimSpace(string(data))
			if name != "" {
				if err := profile.ValidateName(name); err != nil {
					return Result{}, fmt.Errorf("%w: %s contains %q", ErrInvalidDotPaddock, dotPath, name)
				}
				if _, exists := cfg.Profiles[name]; !exists {
					return Result{}, &NotInConfigError{Name: name, Source: fmt.Sprintf(".paddock file at %s", cur)}
				}
				return Result{Profile: name, Rule: RuleDotPaddockFile, Source: cur}, nil
			}
		}
		if cur == stopAt {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}

	// Rule 3: links.json longest-prefix
	if links != nil && len(links.Links) > 0 {
		stopAt := homeStop(pwd)
		bestMatch := ""
		cur := pwd
		for {
			if _, ok := links.Links[cur]; ok {
				if len(cur) > len(bestMatch) {
					bestMatch = cur
				}
			}
			if cur == stopAt {
				break
			}
			parent := filepath.Dir(cur)
			if parent == cur {
				break
			}
			cur = parent
		}
		if bestMatch != "" {
			name := links.Links[bestMatch]
			if _, exists := cfg.Profiles[name]; !exists {
				return Result{}, &NotInConfigError{Name: name, Source: fmt.Sprintf("links.json entry for %s", bestMatch)}
			}
			return Result{Profile: name, Rule: RuleLinksJSON, Source: bestMatch}, nil
		}
	}

	// Rule 4: default profile
	if cfg.DefaultProfile != "" {
		if _, exists := cfg.Profiles[cfg.DefaultProfile]; !exists {
			return Result{}, &NotInConfigError{Name: cfg.DefaultProfile, Source: "default_profile in config.json"}
		}
		return Result{Profile: cfg.DefaultProfile, Rule: RuleDefaultProfile}, nil
	}

	// Rule 5: nothing applies
	return Result{Rule: RuleNone}, nil
}

func homeStop(pwd string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	homeWithSep := home + string(filepath.Separator)
	if pwd == home || strings.HasPrefix(pwd, homeWithSep) {
		return home
	}
	return ""
}
