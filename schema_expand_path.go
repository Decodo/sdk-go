package decodo

import (
	"os"
	"path/filepath"
	"strings"
)

func expandPath(path string) string {
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	abs, _ := filepath.Abs(path)
	return abs
}
