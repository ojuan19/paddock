package shell

import (
	"embed"
	"errors"
)

const (
	StartMarker = "# >>> paddock shell integration >>>"
	EndMarker   = "# <<< paddock shell integration <<<"
)

var ErrUnsupportedShell = errors.New("unsupported shell")

//go:embed scripts/*
var scripts embed.FS

var scriptFiles = map[string]string{
	"bash":       "scripts/bash.sh",
	"zsh":        "scripts/zsh.sh",
	"fish":       "scripts/fish.fish",
	"powershell": "scripts/powershell.ps1",
}

// Script returns the shell integration script for the given shell name.
// Returns ErrUnsupportedShell for unrecognized names.
func Script(name string) (string, error) {
	path, ok := scriptFiles[name]
	if !ok {
		return "", ErrUnsupportedShell
	}
	data, err := scripts.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
