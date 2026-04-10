package troubleshoot

import (
	"testing"

	"github.com/noso-dev/noso/internal/evidence"
)

func TestRuntimeUnhealthyTroubleshoot(t *testing.T) {
	response, ok := Resolve("docker container unhealthy", evidence.NewCollector())
	if !ok {
		t.Fatal("expected runtime unhealthy troubleshoot match")
	}
	if response.IntentID != "troubleshoot_runtime_unhealthy" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestRuntimeImagePullTroubleshoot(t *testing.T) {
	response, ok := Resolve("image pull failed in podman", evidence.NewCollector())
	if !ok {
		t.Fatal("expected runtime image pull troubleshoot match")
	}
	if response.IntentID != "troubleshoot_runtime_image_pull" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}
