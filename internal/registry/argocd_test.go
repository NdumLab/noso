package registry

import (
	"testing"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/pkg/models"
)

func TestArgoCDAppsIntent(t *testing.T) {
	response, err := Resolve("list argocd apps", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_argocd_applications" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestArgoCDVersionIntent(t *testing.T) {
	response, err := Resolve("argocd version", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_argocd_version" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestArgoCDAccountIntent(t *testing.T) {
	response, err := Resolve("show argocd account", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_argocd_account" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestArgoCDAppGetIntent(t *testing.T) {
	response, err := Resolve("show argocd app payments", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_argocd_application" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestArgoCDProjectsAndClustersIntents(t *testing.T) {
	projectResponse, err := Resolve("list argocd projects", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve(projects) error = %v", err)
	}
	if projectResponse.IntentID != "inspect_argocd_projects" {
		t.Fatalf("projects intent = %q", projectResponse.IntentID)
	}

	clusterResponse, err := Resolve("list argocd clusters", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve(clusters) error = %v", err)
	}
	if clusterResponse.IntentID != "inspect_argocd_clusters" {
		t.Fatalf("clusters intent = %q", clusterResponse.IntentID)
	}
}
