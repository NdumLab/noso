package registry

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/pkg/models"
)

func TestSSHConfigIntent(t *testing.T) {
	response, err := Resolve("show ssh config for host app1", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_ssh_config" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if !strings.Contains(response.Command, "ssh -G app1") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestRsyncDryRunIntent(t *testing.T) {
	response, err := Resolve("copy ./dist/ to user@host:/srv/app/", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "preview_rsync_transfer" && response.IntentID != "preview_scp_transfer" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestSSHHostKeyIntent(t *testing.T) {
	response, err := Resolve("show ssh host key for host app1", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !strings.Contains(response.Command, "app1") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestSSHConnectivityIntent(t *testing.T) {
	response, err := Resolve("check ssh connectivity to app1", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_ssh_port_reachability" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if !strings.Contains(response.Command, "app1 22") {
		t.Fatalf("Command = %q", response.Command)
	}
}
