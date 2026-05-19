package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Links struct {
	Version int               `json:"version"`
	Links   map[string]string `json:"links"`
}

func LinksPath() (string, error) {
	dir, err := paddockDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "links.json"), nil
}

func LoadLinks() (*Links, error) {
	path, err := LinksPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Links{Version: CurrentVersion, Links: map[string]string{}}, nil
		}
		return nil, fmt.Errorf("reading links: %w", err)
	}
	var l Links
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("parsing links: %w", err)
	}
	if l.Links == nil {
		l.Links = map[string]string{}
	}
	return &l, nil
}

// RenameProfile rewrites every link value matching oldName to newName and
// returns the affected paths so callers can target follow-up work (e.g.
// updating .paddock files at those paths). Caller is responsible for Save().
func (l *Links) RenameProfile(oldName, newName string) []string {
	if l.Links == nil {
		return nil
	}
	var affected []string
	for path, name := range l.Links {
		if name == oldName {
			l.Links[path] = newName
			affected = append(affected, path)
		}
	}
	return affected
}

func (l *Links) Save() error {
	l.Version = CurrentVersion
	if l.Links == nil {
		l.Links = map[string]string{}
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling links: %w", err)
	}
	path, err := LinksPath()
	if err != nil {
		return err
	}
	if err := WriteFileAtomic(path, data); err != nil {
		return fmt.Errorf("saving links: %w", err)
	}
	return nil
}
