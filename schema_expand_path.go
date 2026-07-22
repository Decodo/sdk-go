package decodo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func expandPath(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expanding path %q: %w", path, err)
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expanding path %q: %w", path, err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving path %q: %w", path, err)
	}
	return abs, nil
}
