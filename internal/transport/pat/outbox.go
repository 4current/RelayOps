package pat

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func OutboxDir(mycall string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	call := strings.ToUpper(strings.TrimSpace(mycall))
	if call == "" {
		return "", fmt.Errorf("empty mycall")
	}

	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "pat", "mailbox", call, "out"), nil
	}

	// Linux default
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "pat", "mailbox", call, "out"), nil
}

func WriteB2F(outboxDir, mid string, b []byte) (string, error) {
	if err := os.MkdirAll(outboxDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(outboxDir, mid+".b2f")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return path, nil
}
