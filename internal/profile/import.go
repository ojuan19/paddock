package profile

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Allowlist (not denylist) of top-level entries from ~/.claude/ to import.
// Allowlist keeps imports forward-safe: new Claude Code internal dirs
// (cache/, history.jsonl, etc.) won't leak into profiles silently.
var importAllowlist = map[string]bool{
	".claude.json":  true,
	"settings.json": true,
	"agents":        true,
	"commands":      true,
	"skills":        true,
	"plugins":       true,
}

// ImportSummary records per-top-level-entry file counts for display.
type ImportSummary struct {
	Files map[string]int // top-level name -> file count copied
}

// ImportFrom copies the allowlisted contents of srcDir into the profile dir
// for `name`. Skips symlinks. Preserves file mode bits. Returns counts per
// top-level entry. The profile dir must already exist (call Create first).
func ImportFrom(srcDir, name string) (ImportSummary, error) {
	sum := ImportSummary{Files: map[string]int{}}
	dstDir, err := Dir(name)
	if err != nil {
		return sum, err
	}

	return sum, filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}

		rel, _ := filepath.Rel(srcDir, path)
		topLevel := rel
		if idx := indexSep(rel); idx >= 0 {
			topLevel = rel[:idx]
		}
		if !importAllowlist[topLevel] {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		info, lerr := os.Lstat(path)
		if lerr != nil {
			return lerr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		dstPath := filepath.Join(dstDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o700)
		}

		if err := copyFile(path, dstPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf("copying %s: %w", rel, err)
		}
		sum.Files[topLevel]++
		return nil
	})
}

func indexSep(s string) int {
	for i, r := range s {
		if r == filepath.Separator || r == '/' {
			return i
		}
	}
	return -1
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
