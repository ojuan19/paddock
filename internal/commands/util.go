package commands

import (
	"sort"

	"github.com/ojuan19/paddock/internal/config"
)

func profileNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
