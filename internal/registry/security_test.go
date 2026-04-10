package registry

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/pkg/models"
)

func TestSELinuxModeIntent(t *testing.T) {
	response, err := Resolve("show selinux mode", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_selinux_mode" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestSELinuxStatusIntent(t *testing.T) {
	response, err := Resolve("show selinux status", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_selinux_status" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestFirewalldRulesIntent(t *testing.T) {
	response, err := Resolve("show firewall rules", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_firewalld_rules" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestFirewalldZonesIntent(t *testing.T) {
	response, err := Resolve("show firewall zones", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_firewalld_zones" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestCertificateIntent(t *testing.T) {
	response, err := Resolve("inspect certificate tls.pem", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_certificate" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if !strings.Contains(response.Command, "tls.pem") {
		t.Fatalf("Command = %q", response.Command)
	}
}
