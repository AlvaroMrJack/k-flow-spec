package discovery

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindWorkspaceRoot walks up the directory tree looking for kfs.yaml
func FindWorkspaceRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(dir, "kfs.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root of the filesystem without finding it
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("kfs.yaml not found in any parent directory")
}
