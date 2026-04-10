package output

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/noso-dev/noso/pkg/models"
)

func TestRenderResponseText(t *testing.T) {
	response := models.Response{
		IntentID:       "inspect_port_listener",
		Command:        "ss -ltnp | grep :8080",
		Explanation:    "Inspect listeners on TCP port 8080.",
		ExpectedOutput: "A matching process and PID if the port is in use.",
		Risk:           "Low",
		Confidence:     "High",
		VerifiedFrom:   []string{"exec.LookPath", "ss --help"},
		NextSteps:      []string{"Run journalctl if the owning service is failing."},
	}

	rendered, err := RenderResponse(response, false, false)
	if err != nil {
		t.Fatalf("RenderResponse() error = %v", err)
	}
	for _, token := range []string{"Command:", "Risk: Low", "Confidence: High", "Next step:"} {
		if !strings.Contains(rendered, token) {
			t.Fatalf("rendered output missing %q", token)
		}
	}
}

func TestRenderResponseQuietSuppressesWarnings(t *testing.T) {
	response := models.Response{
		IntentID:    "test",
		Command:     "df -h",
		Risk:        "Low",
		Confidence:  "High",
		NextSteps:   []string{"next step here"},
		Warnings:    []string{"a warning"},
	}
	rendered, err := RenderResponse(response, false, true)
	if err != nil {
		t.Fatalf("RenderResponse() error = %v", err)
	}
	if strings.Contains(rendered, "Warning:") {
		t.Error("quiet mode should suppress warnings")
	}
	if strings.Contains(rendered, "Next step:") {
		t.Error("quiet mode should suppress next steps")
	}
}

func TestRenderResponseJSON(t *testing.T) {
	response := models.Response{
		IntentID:  "test_json",
		Command:   "df -h",
		Risk:      "Low",
		Confidence: "High",
	}
	rendered, err := RenderResponse(response, true, false)
	if err != nil {
		t.Fatalf("RenderResponse() error = %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(rendered), &out); err != nil {
		t.Fatalf("JSON output is not valid: %v", err)
	}
	if out["intent_id"] != "test_json" {
		t.Errorf("intent_id = %v", out["intent_id"])
	}
}

func TestRenderEnvironmentText(t *testing.T) {
	env := models.Environment{
		PrettyName:     "Red Hat Enterprise Linux 9.7 (Plow)",
		Distro:         "rhel",
		PackageManager: "dnf",
		Shell:          "/bin/bash",
		Commands: map[string]models.CommandInfo{
			"git":    {Exists: true, Path: "/usr/bin/git"},
			"docker": {Exists: false},
		},
	}
	rendered, err := RenderEnvironment(env, false)
	if err != nil {
		t.Fatalf("RenderEnvironment() error = %v", err)
	}
	for _, want := range []string{"Distro: rhel", "Package manager: dnf", "Shell: /bin/bash"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered env missing %q\nGot: %s", want, rendered)
		}
	}
	if !strings.Contains(rendered, "git: /usr/bin/git") {
		t.Errorf("expected git path in output")
	}
	if !strings.Contains(rendered, "docker: missing") {
		t.Errorf("expected docker as missing in output")
	}
}

func TestRenderEnvironmentJSON(t *testing.T) {
	env := models.Environment{
		Distro:         "debian",
		PackageManager: "apt",
	}
	rendered, err := RenderEnvironment(env, true)
	if err != nil {
		t.Fatalf("RenderEnvironment() error = %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(rendered), &out); err != nil {
		t.Fatalf("JSON output is not valid: %v", err)
	}
	if out["distro"] != "debian" {
		t.Errorf("distro = %v", out["distro"])
	}
}
