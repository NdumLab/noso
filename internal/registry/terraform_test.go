package registry

import (
	"testing"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/pkg/models"
)

func TestTerraformVersionIntent(t *testing.T) {
	response, err := Resolve("terraform version", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_terraform_version" {
		t.Fatalf("IntentID = %q, want inspect_terraform_version", response.IntentID)
	}
}

func TestTerraformFmtCheckIntent(t *testing.T) {
	response, err := Resolve("check terraform formatting", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "validate_terraform_formatting" {
		t.Fatalf("IntentID = %q, want validate_terraform_formatting", response.IntentID)
	}
}

func TestTerraformWorkspaceListIntent(t *testing.T) {
	response, err := Resolve("list terraform workspaces", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_terraform_workspaces" {
		t.Fatalf("IntentID = %q, want inspect_terraform_workspaces", response.IntentID)
	}
}

func TestTerraformStateListIntent(t *testing.T) {
	response, err := Resolve("list terraform state", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_terraform_state" {
		t.Fatalf("IntentID = %q, want inspect_terraform_state", response.IntentID)
	}
}

func TestTerraformValidateIntent(t *testing.T) {
	response, err := Resolve("validate terraform", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "validate_terraform_configuration" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestTerraformPlanIntent(t *testing.T) {
	response, err := Resolve("preview terraform plan", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "preview_terraform_plan" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}
