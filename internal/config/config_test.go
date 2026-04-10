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
