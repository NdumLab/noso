package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesEnvOverrides(t *testing.T) {
	t.Setenv("NOSO_MODE", "local-preferred")
	t.Setenv("NOSO_AUDIT_LOG_PATH", filepath.Join(t.TempDir(), "audit.log"))
	t.Setenv("NOSO_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("NOSO_LLM_ENABLED", "true")
	t.Setenv("NOSO_LLM_ENDPOINT", "http://127.0.0.1:15321/v1/interpret")
	t.Setenv("NOSO_LLM_TIMEOUT_MS", "2500")
	t.Setenv("NOSO_LLM_LOG_PATH", filepath.Join(t.TempDir(), "noso-llm.jsonl"))
	t.Setenv("NOSO_TROUBLESHOOT_STATE_PATH", filepath.Join(t.TempDir(), "troubleshoot-state.json"))
	t.Setenv("NOSO_INCIDENT_STATE_PATH", filepath.Join(t.TempDir(), "incident-state.json"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Mode != "local-preferred" {
		t.Fatalf("Mode = %q, want %q", cfg.Mode, "local-preferred")
	}
	if cfg.AuditLogPath == "" {
		t.Fatal("AuditLogPath should not be empty")
	}
	if !cfg.LLMEnabled {
		t.Fatal("LLMEnabled = false, want true")
	}
	if cfg.LLMTimeoutMS != 2500 {
		t.Fatalf("LLMTimeoutMS = %d, want 2500", cfg.LLMTimeoutMS)
	}
	if cfg.LLMLogPath == "" {
		t.Fatal("LLMLogPath should not be empty")
	}
	if cfg.TroubleshootStatePath == "" {
		t.Fatal("TroubleshootStatePath should not be empty")
	}
	if cfg.IncidentStatePath == "" {
		t.Fatal("IncidentStatePath should not be empty")
	}
}

func TestLoadReadsJSONConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := []byte(`{"mode":"strict-local","audit_log_path":"/tmp/noso-audit.log"}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("NOSO_CONFIG", path)
	t.Setenv("NOSO_MODE", "")
	t.Setenv("NOSO_AUDIT_LOG_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AuditLogPath != "/tmp/noso-audit.log" {
		t.Fatalf("AuditLogPath = %q, want /tmp/noso-audit.log", cfg.AuditLogPath)
	}
}

func TestDefaultAuditLogPathPrefersWritableXDGStateHome(t *testing.T) {
	stateHome := filepath.Join(t.TempDir(), "state")
	t.Setenv("XDG_STATE_HOME", stateHome)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))

	path := defaultAuditLogPath()
	want := filepath.Join(stateHome, "noso", "audit.log")
	if path != want {
		t.Fatalf("defaultAuditLogPath() = %q, want %q", path, want)
	}
}

func TestAuditPathUsableWithWritableDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "noso", "audit.log")
	if !AuditPathUsable(path) {
		t.Fatal("AuditPathUsable() = false, want true")
	}
}

func TestValidateAcceptsKnownModes(t *testing.T) {
	for _, mode := range []string{"strict-local", "local-preferred"} {
		cfg := Config{Mode: mode, AuditLogPath: "/tmp/test.log"}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() with mode %q returned unexpected error: %v", mode, err)
		}
	}
}

func TestValidateRejectsUnknownMode(t *testing.T) {
	cfg := Config{Mode: "turbo-mode", AuditLogPath: "/tmp/test.log"}
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() with invalid mode should return an error")
	}
}

func TestLoadRejectsInvalidModeFromEnv(t *testing.T) {
	t.Setenv("NOSO_MODE", "not-a-real-mode")
	t.Setenv("NOSO_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	_, err := Load()
	if err == nil {
		t.Error("Load() with invalid NOSO_MODE should return an error")
	}
}

func TestDefaultAuditLogPathFallsBackWhenHomeStateIsNotWritable(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	home := filepath.Join(t.TempDir(), "home")
	if err := os.MkdirAll(filepath.Join(home, ".local"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.Chmod(filepath.Join(home, ".local"), 0o555); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	t.Setenv("HOME", home)

	path := defaultAuditLogPath()
	want := filepath.Join(os.TempDir(), "noso", "audit.log")
	if path != want {
		t.Fatalf("defaultAuditLogPath() = %q, want %q", path, want)
	}
}

func TestValidateRejectsInvalidLLMEndpoint(t *testing.T) {
	cfg := Config{
		Mode:         "strict-local",
		AuditLogPath: "/tmp/test.log",
		LLMEnabled:   true,
		LLMEndpoint:  "://bad",
		LLMTimeoutMS: 1000,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() with invalid llm endpoint should fail")
	}
}

func TestLoadRejectsInvalidLLMTimeout(t *testing.T) {
	t.Setenv("NOSO_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("NOSO_LLM_ENABLED", "true")
	t.Setenv("NOSO_LLM_TIMEOUT_MS", "abc")
	if _, err := Load(); err == nil {
		t.Fatal("Load() with invalid NOSO_LLM_TIMEOUT_MS should fail")
	}
}
