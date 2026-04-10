package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Mode         string `json:"mode"`
	AuditLogPath string `json:"audit_log_path"`
}

var validModes = map[string]bool{
	"strict-local":    true,
	"local-preferred": true,
}

// Validate returns an error if the config contains unsupported field values.
func (c Config) Validate() error {
	if !validModes[c.Mode] {
		return fmt.Errorf("invalid mode %q: must be one of: strict-local, local-preferred", c.Mode)
	}
	return nil
}

func Load() (Config, error) {
	cfg := defaultConfig()

	if filePath := configFilePath(); filePath != "" {
		if data, err := os.ReadFile(filePath); err == nil {
			if err := json.Unmarshal(data, &cfg); err != nil {
				return Config{}, fmt.Errorf("config file is not valid JSON: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return Config{}, err
		}
	}

	if v := os.Getenv("NOSO_MODE"); v != "" {
		cfg.Mode = v
	}
	if v := os.Getenv("NOSO_AUDIT_LOG_PATH"); v != "" {
		cfg.AuditLogPath = v
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Mode:         "strict-local",
		AuditLogPath: defaultAuditLogPath(),
	}
}

func defaultAuditLogPath() string {
	candidates := []string{}
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		candidates = append(candidates, filepath.Join(stateHome, "noso", "audit.log"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".local", "state", "noso", "audit.log"))
	}
	candidates = append(candidates, filepath.Join(os.TempDir(), "noso", "audit.log"))

	for _, candidate := range candidates {
		if AuditPathUsable(candidate) {
			return candidate
		}
	}

	return filepath.Join(os.TempDir(), "noso", "audit.log")
}

func AuditPathUsable(path string) bool {
	dir := filepath.Dir(path)
	// Use 0o700 so that any newly-created probe directory is already private.
	// The logger will enforce this again on first write.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false
	}

	f, err := os.CreateTemp(dir, ".noso-audit-check-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

func configFilePath() string {
	if v := os.Getenv("NOSO_CONFIG"); v != "" {
		return v
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "noso", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "noso", "config.json")
}
