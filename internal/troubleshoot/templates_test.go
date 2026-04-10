package troubleshoot

import (
	"testing"

	"github.com/noso-dev/noso/internal/evidence"
)

func TestServiceTroubleshoot(t *testing.T) {
	response, ok := Resolve("nginx is not starting", evidence.NewCollector())
	if !ok {
		t.Fatal("expected service troubleshoot match")
	}
	if response.IntentID != "troubleshoot_service_failure" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if len(response.NextSteps) == 0 {
		t.Fatal("expected next steps")
	}
}

func TestDiskTroubleshoot(t *testing.T) {
	response, ok := Resolve("disk full on /var", evidence.NewCollector())
	if !ok {
		t.Fatal("expected disk troubleshoot match")
	}
	if response.Command != "df -h" {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestK8sCrashLoopTroubleshoot(t *testing.T) {
	response, ok := Resolve("pod is in CrashLoopBackOff", evidence.NewCollector())
	if !ok {
		t.Fatal("expected k8s crashloop match")
	}
	if response.IntentID != "troubleshoot_k8s_crashloopbackoff" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if len(response.NextSteps) == 0 {
		t.Fatal("expected next steps")
	}
}

func TestK8sImagePullBackoffTroubleshoot(t *testing.T) {
	response, ok := Resolve("pod stuck with ImagePullBackOff", evidence.NewCollector())
	if !ok {
		t.Fatal("expected imagepullbackoff match")
	}
	if response.IntentID != "troubleshoot_k8s_imagepullbackoff" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if len(response.NextSteps) == 0 {
		t.Fatal("expected next steps")
	}
}

func TestK8sPendingPodTroubleshoot(t *testing.T) {
	response, ok := Resolve("pod pending in namespace prod", evidence.NewCollector())
	if !ok {
		t.Fatal("expected pending pod match")
	}
	if response.IntentID != "troubleshoot_k8s_pending_pod" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestRuntimeStartFailureTroubleshoot(t *testing.T) {
	response, ok := Resolve("docker container failed to start", evidence.NewCollector())
	if !ok {
		t.Fatal("expected runtime start failure match")
	}
	if response.IntentID != "troubleshoot_runtime_start_failure" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestNetworkConnectTroubleshoot(t *testing.T) {
	response, ok := Resolve("connection refused to port 8080", evidence.NewCollector())
	if !ok {
		t.Fatal("expected network connect match")
	}
	if response.IntentID != "troubleshoot_network_connectivity" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if len(response.NextSteps) == 0 {
		t.Fatal("expected next steps")
	}
}

func TestTroubleshootNoMatch(t *testing.T) {
	_, ok := Resolve("show disk free space", evidence.NewCollector())
	if ok {
		t.Error("expected no match for unrelated query")
	}
}

func TestRuntimeFromQueryPodman(t *testing.T) {
	if got := runtimeFromQuery("podman container unhealthy"); got != "podman" {
		t.Errorf("runtimeFromQuery(podman) = %q, want podman", got)
	}
}

func TestRuntimeFromQueryContainerd(t *testing.T) {
	if got := runtimeFromQuery("containerd not starting"); got != "containerd" {
		t.Errorf("runtimeFromQuery(containerd) = %q, want containerd", got)
	}
}

func TestRuntimeFromQueryNerdctl(t *testing.T) {
	if got := runtimeFromQuery("containerd nerdctl failed to start"); got != "nerdctl" {
		t.Errorf("runtimeFromQuery(nerdctl) = %q, want nerdctl", got)
	}
}

func TestRuntimeFromQueryCrictl(t *testing.T) {
	if got := runtimeFromQuery("containerd crictl failed to start"); got != "crictl" {
		t.Errorf("runtimeFromQuery(crictl) = %q, want crictl", got)
	}
}

func TestRuntimeFromQueryCtr(t *testing.T) {
	if got := runtimeFromQuery("containerd ctr failed to start"); got != "ctr" {
		t.Errorf("runtimeFromQuery(ctr) = %q, want ctr", got)
	}
}

func TestRuntimeFromQueryDefaultDocker(t *testing.T) {
	if got := runtimeFromQuery("container unhealthy"); got != "docker" {
		t.Errorf("runtimeFromQuery(default) = %q, want docker", got)
	}
}
