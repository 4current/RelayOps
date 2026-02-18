package pat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	MyCall string `json:"mycall"`
}

func LoadConfig() (*Config, string, error) {
	if p := os.Getenv("PAT_CONFIG"); p != "" {
		cfg, err := readConfig(p)
		return cfg, p, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", err
	}

	// Linux
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		p := filepath.Join(xdg, "pat", "config.json")
		if exists(p) {
			cfg, err := readConfig(p)
			return cfg, p, err
		}
	} else {
		p := filepath.Join(home, ".config", "pat", "config.json")
		if exists(p) {
			cfg, err := readConfig(p)
			return cfg, p, err
		}
	}

	// macOS (your current setup)
	if runtime.GOOS == "darwin" {
		p := filepath.Join(home, "Library", "Application Support", "pat", "config.json")
		if exists(p) {
			cfg, err := readConfig(p)
			return cfg, p, err
		}
	}

	return nil, "", fmt.Errorf("pat config.json not found (set PAT_CONFIG to override)")
}

func readConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.MyCall == "" {
		return nil, fmt.Errorf("pat config %s: mycall is empty", path)
	}
	return &cfg, nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
