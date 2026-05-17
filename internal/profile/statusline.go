package profile

import (
	"os"
	"strings"
)

// HasExistingStatusLine returns true if the file at path contains a top-level
// "statusLine" key in its JSON. Substring match — false positives only matter
// in pathological JSON.
func HasExistingStatusLine(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), `"statusLine"`)
}
