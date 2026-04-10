package registry

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/evidence"
)

func TestRuntimeStatusIntentDocker(t *testing.T) {
	response, err := runtimeStatusIntent("docker status", evidence.NewCollector())
	if err != nil {
		t.Fatalf("runtimeStatusIntent() error = %v", err)
	}
	if response.Command != "systemctl status docker --no-pager -l" {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestRuntimeVersionIntentDocker(t *testing.T) {
	response, err := runtimeVersionIntent("docker version", evidence.NewCollector())
	if err != nil {
		t.Fatalf("runtimeVersionIntent() error = %v", err)
	}
	if response.IntentID != "inspect_runtime_version" {
		t.Fatalf("IntentID = %q, want inspect_runtime_version", response.IntentID)
	}
	if !strings.Contains(response.Command, "docker") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestRuntimePsIntentDocker(t *testing.T) {
	response, err := runtimePsIntent("list docker containers", evidence.NewCollector())
	if err != nil {
		t.Fatalf("runtimePsIntent() error = %v", err)
	}
	if response.IntentID != "inspect_runtime_containers" {
		t.Fatalf("IntentID = %q, want inspect_runtime_containers", response.IntentID)
	}
	if !strings.Contains(response.Command, "docker ps") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestRuntimeImagesIntentDocker(t *testing.T) {
	response, err := runtimeImagesIntent("list docker images", evidence.NewCollector())
	if err != nil {
		t.Fatalf("runtimeImagesIntent() error = %v", err)
	}
	if response.IntentID != "inspect_runtime_images" {
		t.Fatalf("IntentID = %q, want inspect_runtime_images", response.IntentID)
	}
}

func TestRuntimeInspectIntentDocker(t *testing.T) {
	response, err := runtimeInspectIntent("inspect docker container myapp", evidence.NewCollector())
	if err != nil {
		t.Fatalf("runtimeInspectIntent() error = %v", err)
	}
	if response.IntentID != "inspect_runtime_container" {
		t.Fatalf("IntentID = %q, want inspect_runtime_container", response.IntentID)
	}
	if !strings.Contains(response.Command, "myapp") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestRuntimeLogsIntentPodman(t *testing.T) {
	response, err := runtimeLogsIntent("podman logs web", evidence.NewCollector())
	if err != nil {
		t.Fatalf("runtimeLogsIntent() error = %v", err)
	}
	if !strings.Contains(response.Command, "podman logs --tail 100") {
		t.Fatalf("Command = %q", response.Command)
	}
}
