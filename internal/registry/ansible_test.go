package registry

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/pkg/models"
)

func TestAnsibleVersionIntent(t *testing.T) {
	response, err := Resolve("ansible version", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_ansible_version" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestAnsibleSyntaxCheckIntent(t *testing.T) {
	response, err := Resolve("syntax check playbook site.yml", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "validate_ansible_playbook_syntax" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if !strings.Contains(response.Command, "site.yml") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestAnsibleInventoryIntent(t *testing.T) {
	response, err := Resolve("list ansible inventory", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_ansible_inventory" {
		t.Fatalf("IntentID = %q, want inspect_ansible_inventory", response.IntentID)
	}
}

func TestAnsibleCheckModeIntent(t *testing.T) {
	response, err := Resolve("dry run ansible playbook deploy.yml", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "preview_ansible_check_mode" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}
