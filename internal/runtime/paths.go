package runtime

import (
	"os"
	"path/filepath"
)

// AppDir returns the default per-user runtime directory, e.g. ~/.relayops
func AppDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".relayops"), nil
}

func EnsureAppDir() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func DBPath() (string, error) {
	dir, err := EnsureAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "relayops.db"), nil
}
