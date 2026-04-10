package registry

import (
	"strings"
	"testing"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/pkg/models"
)

func TestHelmReposIntent(t *testing.T) {
	response, err := Resolve("list helm repos", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_helm_repos" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestHelmStatusIntent(t *testing.T) {
	response, err := Resolve("show helm status for release api in namespace prod", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_helm_release_status" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if !strings.Contains(response.Command, "helm status api -n prod") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestHelmVersionIntent(t *testing.T) {
	response, err := Resolve("helm version", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_helm_version" {
		t.Fatalf("IntentID = %q, want inspect_helm_version", response.IntentID)
	}
}

func TestHelmReleasesIntent(t *testing.T) {
	response, err := Resolve("list helm releases", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_helm_releases" {
		t.Fatalf("IntentID = %q, want inspect_helm_releases", response.IntentID)
	}
}

func TestHelmHistoryIntent(t *testing.T) {
	response, err := Resolve("show helm history for release myapp", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_helm_release_history" {
		t.Fatalf("IntentID = %q, want inspect_helm_release_history", response.IntentID)
	}
	if !strings.Contains(response.Command, "myapp") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestHelmValuesIntent(t *testing.T) {
	response, err := Resolve("show helm values for release myapp", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_helm_values" {
		t.Fatalf("IntentID = %q, want inspect_helm_values", response.IntentID)
	}
	if !strings.Contains(response.Command, "myapp") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestHelmTemplateIntent(t *testing.T) {
	response, err := Resolve("preview helm template for chart ingress-nginx", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "preview_helm_template" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}
