package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Mode                  string `json:"mode"`
	AuditLogPath          string `json:"audit_log_path"`
	LLMEnabled            bool   `json:"llm_enabled"`
	LLMEndpoint           string `json:"llm_endpoint"`
	LLMTimeoutMS          int    `json:"llm_timeout_ms"`
	LLMLogPath            string `json:"llm_log_path"`
	TroubleshootStatePath string `json:"troubleshoot_state_path"`
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
	if c.LLMEnabled {
		if _, err := url.ParseRequestURI(c.LLMEndpoint); err != nil {
			return fmt.Errorf("invalid llm endpoint %q: %w", c.LLMEndpoint, err)
		}
		if c.LLMTimeoutMS <= 0 {
			return fmt.Errorf("invalid llm timeout %d: must be greater than zero", c.LLMTimeoutMS)
		}
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
	if v := os.Getenv("NOSO_LLM_ENABLED"); v != "" {
		cfg.LLMEnabled = v == "1" || stringsEqualFold(v, "true") || stringsEqualFold(v, "yes")
	}
	if v := os.Getenv("NOSO_LLM_ENDPOINT"); v != "" {
		cfg.LLMEndpoint = v
	}
	if v := os.Getenv("NOSO_LLM_TIMEOUT_MS"); v != "" {
		timeout, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid NOSO_LLM_TIMEOUT_MS %q: %w", v, err)
		}
		cfg.LLMTimeoutMS = timeout
	}
	if v := os.Getenv("NOSO_LLM_LOG_PATH"); v != "" {
		cfg.LLMLogPath = v
	}
	if v := os.Getenv("NOSO_TROUBLESHOOT_STATE_PATH"); v != "" {
		cfg.TroubleshootStatePath = v
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Mode:                  "strict-local",
		AuditLogPath:          defaultAuditLogPath(),
		LLMEnabled:            false,
		LLMEndpoint:           "http://127.0.0.1:15321/v1/interpret",
		LLMTimeoutMS:          1500,
		LLMLogPath:            "",
		TroubleshootStatePath: defaultTroubleshootStatePath(),
	}
}

func stringsEqualFold(a, b string) bool {
	return len(a) == len(b) && strings.EqualFold(a, b)
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

func defaultTroubleshootStatePath() string {
	candidates := []string{}
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		candidates = append(candidates, filepath.Join(stateHome, "noso", "troubleshoot-state.json"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".local", "state", "noso", "troubleshoot-state.json"))
	}
	candidates = append(candidates, filepath.Join(os.TempDir(), "noso", "troubleshoot-state.json"))

	for _, candidate := range candidates {
		if AuditPathUsable(candidate) {
			return candidate
		}
	}
	return filepath.Join(os.TempDir(), "noso", "troubleshoot-state.json")
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
