package registry

import (
	"testing"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/pkg/models"
)

func TestAWSIdentityIntent(t *testing.T) {
	response, err := Resolve("show aws identity", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_aws_identity" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestAWSProfilesIntent(t *testing.T) {
	response, err := Resolve("list aws profiles", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_aws_profiles" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestAzureAccountIntent(t *testing.T) {
	response, err := Resolve("show azure account", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_azure_account" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestAzureSubscriptionsIntent(t *testing.T) {
	response, err := Resolve("list azure subscriptions", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_azure_subscriptions" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestGCloudProjectIntent(t *testing.T) {
	response, err := Resolve("show gcloud project", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_gcloud_project" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestGCloudAccountIntent(t *testing.T) {
	response, err := Resolve("show gcloud account", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_gcloud_account" {
		t.Fatalf("IntentID = %q, want inspect_gcloud_account", response.IntentID)
	}
}

func TestCloudVersionIntents(t *testing.T) {
	cases := map[string]string{
		"aws version":     "inspect_aws_version",
		"azure version":   "inspect_azure_version",
		"gcloud version":  "inspect_gcloud_version",
	}

	for query, intentID := range cases {
		response, err := Resolve(query, models.Environment{}, evidence.NewCollector())
		if err != nil {
			t.Fatalf("Resolve(%q) error = %v", query, err)
		}
		if response.IntentID != intentID {
			t.Fatalf("Resolve(%q) intent = %q, want %q", query, response.IntentID, intentID)
		}
	}
}
