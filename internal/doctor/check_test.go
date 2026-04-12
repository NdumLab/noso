package doctor

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/config"
	"github.com/NdumLab/noso/pkg/models"
)

func allCoreCommandsPresent() map[string]models.CommandInfo {
	return map[string]models.CommandInfo{
		"bash":      {Exists: true},
		"systemctl": {Exists: true},
		"ss":        {Exists: true},
		"find":      {Exists: true},
		"git":       {Exists: true},
	}
}

func TestCheckWarnsForMissingCoreCommands(t *testing.T) {
	cfg := config.Config{Mode: "strict-local", AuditLogPath: filepath.Join(t.TempDir(), "audit.log")}
	cmds := allCoreCommandsPresent()
	cmds["git"] = models.CommandInfo{Exists: false}
	env := models.Environment{
		OSID:      "rhel",
		VersionID: "9.7",
		Distro:    "rhel",
		IsRHEL9:   true,
		Commands:  cmds,
	}

	response := Check(cfg, env)
	if len(response.Warnings) == 0 {
		t.Fatal("expected warnings for missing core commands")
	}
	if !strings.Contains(strings.Join(response.Warnings, " "), "git") {
		t.Fatalf("Warnings = %v", response.Warnings)
	}
}

func TestCheckHealthyKnownDistroHost(t *testing.T) {
	for _, distro := range []string{"rhel", "debian", "fedora", "suse", "arch"} {
		t.Run(distro, func(t *testing.T) {
			cfg := config.Config{Mode: "strict-local", AuditLogPath: filepath.Join(t.TempDir(), "audit.log")}
			env := models.Environment{
				OSID:     distro,
				Distro:   distro,
				IsRHEL9:  distro == "rhel",
				Commands: allCoreCommandsPresent(),
			}
			response := Check(cfg, env)
			if len(response.Warnings) != 0 {
				t.Fatalf("distro=%s: unexpected warnings: %v", distro, response.Warnings)
			}
			if !strings.Contains(response.Explanation, "No blocking issues") {
				t.Fatalf("distro=%s: Explanation = %q", distro, response.Explanation)
			}
		})
	}
}

func TestCheckWarnsForUnknownDistro(t *testing.T) {
	cfg := config.Config{Mode: "strict-local", AuditLogPath: filepath.Join(t.TempDir(), "audit.log")}
	env := models.Environment{
		OSID:     "mylinux",
		Distro:   "unknown",
		Commands: allCoreCommandsPresent(),
	}

	response := Check(cfg, env)
	if len(response.Warnings) == 0 {
		t.Fatal("expected warning for unknown distro family")
	}
	combined := strings.Join(response.Warnings, " ")
	if !strings.Contains(combined, "not yet fully validated") {
		t.Fatalf("Warnings = %v", response.Warnings)
	}
}

// TestCheckLegacyFallback verifies that environments without a Distro field
// still get a warning when IsRHEL9 is false, preserving backward compatibility.
func TestCheckLegacyFallback(t *testing.T) {
	cfg := config.Config{Mode: "strict-local", AuditLogPath: filepath.Join(t.TempDir(), "audit.log")}
	env := models.Environment{
		OSID:      "ubuntu",
		VersionID: "24.04",
		Distro:    "", // empty — old-style environment
		IsRHEL9:   false,
		Commands:  allCoreCommandsPresent(),
	}

	response := Check(cfg, env)
	if len(response.Warnings) == 0 {
		t.Fatal("expected warning for non-RHEL9 host with empty Distro")
	}
	if !strings.Contains(strings.Join(response.Warnings, " "), "ubuntu") {
		t.Fatalf("Warnings = %v", response.Warnings)
	}
}

func TestCheckHealthyLLMEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","provider":"heuristic","model":"heuristic-local"}`))
	}))
	defer server.Close()

	cfg := config.Config{
		Mode:         "strict-local",
		AuditLogPath: filepath.Join(t.TempDir(), "audit.log"),
		LLMEnabled:   true,
		LLMEndpoint:  server.URL + "/v1/interpret",
		LLMTimeoutMS: 500,
	}
	env := models.Environment{
		OSID:     "rhel",
		Distro:   "rhel",
		IsRHEL9:  true,
		Commands: allCoreCommandsPresent(),
	}

	response := Check(cfg, env)
	if len(response.Warnings) != 0 {
		t.Fatalf("Warnings = %v, want none", response.Warnings)
	}
	if !strings.Contains(response.Explanation, "Local LLM fallback is healthy") {
		t.Fatalf("Explanation = %q", response.Explanation)
	}
}

func TestCheckWarnsForUnavailableLLMEndpoint(t *testing.T) {
	cfg := config.Config{
		Mode:         "strict-local",
		AuditLogPath: filepath.Join(t.TempDir(), "audit.log"),
		LLMEnabled:   true,
		LLMEndpoint:  "http://127.0.0.1:1/v1/interpret",
		LLMTimeoutMS: 100,
	}
	env := models.Environment{
		OSID:     "rhel",
		Distro:   "rhel",
		IsRHEL9:  true,
		Commands: allCoreCommandsPresent(),
	}

	response := Check(cfg, env)
	if len(response.Warnings) == 0 {
		t.Fatal("expected LLM warning")
	}
	if !strings.Contains(strings.Join(response.Warnings, " "), "local llm") {
		t.Fatalf("Warnings = %v", response.Warnings)
	}
	if len(response.NextSteps) == 0 {
		t.Fatal("expected next step for LLM issue")
	}
}
