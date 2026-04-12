package troubleshoot

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/NdumLab/noso/pkg/models"
)

func TestEnrichWithRunnerServiceFailure(t *testing.T) {
	response := models.Response{
		IntentID:    "troubleshoot_plan",
		Command:     "systemctl status worker2 --no-pager -l",
		Explanation: "Built a ranked, read-only troubleshoot plan.",
		Risk:        "Low",
		Confidence:  "Medium",
	}

	runner := func(_ context.Context, command string) (string, error) {
		switch command {
		case "systemctl status worker2 --no-pager -l":
			return "Active: failed (Result: exit-code)\nResult: exit-code", nil
		case "journalctl -u worker2 -n 50 --no-pager":
			return "worker2[123]: permission denied while reading /etc/worker2.conf", nil
		default:
			return "", fmt.Errorf("unexpected command %q", command)
		}
	}

	updated := enrichWithRunner(response, runner)
	if !containsPrefix(updated.Findings, "Live service evidence: The unit is in a failed state.") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	if !containsPrefix(updated.Findings, "Journal evidence: The pasted log output contains") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	if !contains(updated.VerifiedFrom, "live:systemctl status worker2 --no-pager -l") {
		t.Fatalf("VerifiedFrom = %#v", updated.VerifiedFrom)
	}
	if !contains(updated.VerifiedFrom, "live:journalctl -u worker2 -n 50 --no-pager") {
		t.Fatalf("VerifiedFrom = %#v", updated.VerifiedFrom)
	}
	if !containsPrefix(updated.NextSteps, "Evidence follow-up: Run `journalctl -u <service> -n 50 --no-pager`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
}

func TestEnrichWithRunnerSkipsNonStatusCommands(t *testing.T) {
	response := models.Response{
		IntentID:    "troubleshoot_plan",
		Command:     "df -h",
		Explanation: "Container hypothesis.",
		Risk:        "Low",
	}
	updated := enrichWithRunner(response, func(_ context.Context, command string) (string, error) {
		t.Fatalf("runner called unexpectedly for %q", command)
		return "", nil
	})
	if updated.Explanation != response.Explanation {
		t.Fatalf("Explanation = %q, want %q", updated.Explanation, response.Explanation)
	}
}

func TestEnrichWithRunnerRuntimeFailure(t *testing.T) {
	response := models.Response{
		IntentID:    "troubleshoot_plan",
		Command:     "podman ps -a",
		Explanation: "Built a ranked, read-only troubleshoot plan.",
		Risk:        "Low",
		NextSteps: []string{
			"4. Container hypothesis (0.54): run `podman logs --tail 100 worker2`. Why: container logs are the fastest follow-up once the runtime hypothesis is confirmed",
		},
	}

	runner := func(_ context.Context, command string) (string, error) {
		switch command {
		case "podman ps -a":
			return "CONTAINER ID  IMAGE   COMMAND   CREATED   STATUS                     NAMES\nabc123 app app 1m ago Exited (1) 10 seconds ago worker2", nil
		case "podman logs --tail 100 worker2":
			return "error: failed to bind socket: permission denied", nil
		default:
			return "", fmt.Errorf("unexpected command %q", command)
		}
	}

	updated := enrichWithRunner(response, runner)
	if !containsPrefix(updated.Findings, "Runtime evidence: The runtime container list shows non-running or unhealthy containers.") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	if !containsPrefix(updated.Findings, "Runtime log evidence: The pasted log output contains") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	if !contains(updated.VerifiedFrom, "live:podman ps -a") || !contains(updated.VerifiedFrom, "live:podman logs --tail 100 worker2") {
		t.Fatalf("VerifiedFrom = %#v", updated.VerifiedFrom)
	}
}

func TestEnrichWithRunnerKubernetesFailure(t *testing.T) {
	response := models.Response{
		IntentID:    "troubleshoot_plan",
		Command:     "kubectl describe pod -n prod web-7c5c",
		Explanation: "Built a ranked, read-only troubleshoot plan.",
		Risk:        "Low",
	}

	runner := func(_ context.Context, command string) (string, error) {
		switch command {
		case "kubectl describe pod -n prod web-7c5c":
			return "Containers:\n  api:\n    Container ID: containerd://123\nState: Waiting\nReason: CrashLoopBackOff\nEvents:\nWarning BackOff Back-off restarting failed container", nil
		case "kubectl logs -n prod web-7c5c -c api --tail=100":
			return "panic: failed to connect to database: connection to server at db.internal port 5432 failed", nil
		default:
			return "", fmt.Errorf("unexpected command %q", command)
		}
	}

	updated := enrichWithRunner(response, runner)
	if !containsPrefix(updated.Findings, "Kubernetes evidence: The pasted pod description contains failure signals") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	if !containsPrefix(updated.Findings, "Kubernetes log evidence: The pasted log output contains") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	if !contains(updated.VerifiedFrom, "live:kubectl describe pod -n prod web-7c5c") || !contains(updated.VerifiedFrom, "live:kubectl logs -n prod web-7c5c -c api --tail=100") {
		t.Fatalf("VerifiedFrom = %#v", updated.VerifiedFrom)
	}
	if !containsPrefix(updated.NextSteps, "Evidence follow-up: Run `kubectl logs -n prod web-7c5c -c api --tail=100`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if !containsPrefix(updated.NextSteps, "Evidence follow-up: Run `kubectl get events -n prod --sort-by=.metadata.creationTimestamp`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if !containsPrefix(updated.NextSteps, "Evidence follow-up: Run `dig +short db.internal`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if !containsPrefix(updated.NextSteps, "Evidence follow-up: Run `nc -vz db.internal 5432`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if updated.ContainerHint != "api" {
		t.Fatalf("ContainerHint = %q, want api", updated.ContainerHint)
	}
}

func TestEnrichWithRunnerKubernetesPodsFailure(t *testing.T) {
	response := models.Response{
		IntentID:    "troubleshoot_k8s_crashloopbackoff",
		Command:     "kubectl get pods -A",
		Explanation: "Start by identifying which pod is crashing.",
		Risk:        "Low",
	}

	runner := func(_ context.Context, command string) (string, error) {
		switch command {
		case "kubectl get pods -A":
			return "NAMESPACE NAME READY STATUS RESTARTS AGE\nprod web-7c5c 0/1 CrashLoopBackOff 8 10m", nil
		case "kubectl describe pod -n prod web-7c5c":
			return "Containers:\n  api:\n    Container ID: containerd://123\nState: Waiting\nReason: CrashLoopBackOff\nEvents:\nWarning BackOff Back-off restarting failed container", nil
		case "kubectl logs -n prod web-7c5c -c api --tail=100":
			return "panic: failed to connect to database: connection to server at db.internal port 5432 failed", nil
		default:
			return "", fmt.Errorf("unexpected command %q", command)
		}
	}

	updated := enrichWithRunner(response, runner)
	if !containsPrefix(updated.Findings, "Kubernetes evidence: At least one pod is not healthy") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	if !containsPrefix(updated.Findings, "Kubernetes describe evidence: The pasted pod description contains failure signals") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	if !containsPrefix(updated.Findings, "Kubernetes log evidence: The pasted log output contains") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	if !containsPrefix(updated.NextSteps, "Evidence follow-up: Run `kubectl describe pod -n prod web-7c5c`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if !containsPrefix(updated.NextSteps, "Evidence follow-up: Run `kubectl logs -n prod web-7c5c -c api --previous`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if !containsPrefix(updated.NextSteps, "Evidence follow-up: Run `kubectl get events -n prod --sort-by=.metadata.creationTimestamp`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if !containsPrefix(updated.NextSteps, "Evidence follow-up: Run `dig +short db.internal`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if !containsPrefix(updated.NextSteps, "Evidence follow-up: Run `nc -vz db.internal 5432`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if updated.ContainerHint != "api" {
		t.Fatalf("ContainerHint = %q, want api", updated.ContainerHint)
	}
}

func TestEnrichWithRunnerKubernetesMissingTool(t *testing.T) {
	response := models.Response{
		IntentID:    "troubleshoot_k8s_crashloopbackoff",
		Command:     "kubectl get pods -A",
		Explanation: "Start by identifying which pod is crashing.",
		Risk:        "Low",
	}

	runner := func(_ context.Context, command string) (string, error) {
		return "bash: kubectl: command not found", nil
	}

	updated := enrichWithRunner(response, runner)
	for _, finding := range updated.Findings {
		if strings.Contains(finding, "Kubernetes evidence:") {
			t.Fatalf("Findings = %#v", updated.Findings)
		}
	}
	if !containsPrefix(updated.Warnings, "live kubernetes probe unavailable:") {
		t.Fatalf("Warnings = %#v", updated.Warnings)
	}
}

func TestEnrichWithRunnerKubernetesUsesContainerHintFromEvents(t *testing.T) {
	response := models.Response{
		IntentID:    "troubleshoot_plan",
		Command:     "kubectl describe pod -n prod web-7c5c",
		Explanation: "Built a ranked, read-only troubleshoot plan.",
		Risk:        "Low",
	}

	runner := func(_ context.Context, command string) (string, error) {
		switch command {
		case "kubectl describe pod -n prod web-7c5c":
			return "State: Waiting\nReason: CrashLoopBackOff\nEvents:\n  Warning  BackOff  kubelet  Back-off restarting failed container api in pod web-7c5c_prod(1234)", nil
		case "kubectl logs -n prod web-7c5c -c api --tail=100":
			return "panic: failed to connect to database: connection to server at db.internal port 5432 failed", nil
		default:
			return "", fmt.Errorf("unexpected command %q", command)
		}
	}

	updated := enrichWithRunner(response, runner)
	if !contains(updated.VerifiedFrom, "live:kubectl logs -n prod web-7c5c -c api --tail=100") {
		t.Fatalf("VerifiedFrom = %#v", updated.VerifiedFrom)
	}
	if updated.ContainerHint != "api" {
		t.Fatalf("ContainerHint = %q, want api", updated.ContainerHint)
	}
}

func TestEnrichWithRunnerKubernetesEventsUsesContainerHint(t *testing.T) {
	response := models.Response{
		IntentID:    "inspect_k8s_events",
		Command:     "kubectl get events -n prod --sort-by=.metadata.creationTimestamp",
		Explanation: "Shows recent cluster events in chronological order.",
		Risk:        "Low",
	}

	runner := func(_ context.Context, command string) (string, error) {
		switch command {
		case "kubectl get events -n prod --sort-by=.metadata.creationTimestamp":
			return "LAST SEEN TYPE REASON OBJECT MESSAGE\n10s Warning BackOff pod/web-7c5c Back-off restarting failed container api in pod web-7c5c_prod(1234)", nil
		case "kubectl logs -n prod web-7c5c -c api --previous":
			return "panic: failed to connect to database: connection to server at db.internal port 5432 failed", nil
		default:
			return "", fmt.Errorf("unexpected command %q", command)
		}
	}

	updated := enrichWithRunner(response, runner)
	if !containsPrefix(updated.Findings, "Kubernetes event evidence: The pasted Kubernetes event output contains failure signals") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	if !containsPrefix(updated.Findings, "Kubernetes event log evidence: The pasted log output contains") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	if !contains(updated.VerifiedFrom, "live:kubectl logs -n prod web-7c5c -c api --previous") {
		t.Fatalf("VerifiedFrom = %#v", updated.VerifiedFrom)
	}
	if !containsPrefix(updated.NextSteps, "Evidence follow-up: Run `kubectl logs -n prod web-7c5c -c api --previous`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if updated.ContainerHint != "api" {
		t.Fatalf("ContainerHint = %q, want api", updated.ContainerHint)
	}
}

func TestEnrichWithRunnerKubernetesEventsImagePullDoesNotProbeLogs(t *testing.T) {
	response := models.Response{
		IntentID:    "inspect_k8s_events",
		Command:     "kubectl get events -n prod --sort-by=.metadata.creationTimestamp",
		Explanation: "Shows recent cluster events in chronological order.",
		Risk:        "Low",
	}

	runner := func(_ context.Context, command string) (string, error) {
		switch command {
		case "kubectl get events -n prod --sort-by=.metadata.creationTimestamp":
			return "LAST SEEN TYPE REASON OBJECT MESSAGE\n12s Warning Failed pod/api-6d8f Failed to pull image \"ghcr.io/example/app:bad\": rpc error\n11s Warning ImagePullBackOff pod/api-6d8f Back-off pulling image \"ghcr.io/example/app:bad\"", nil
		default:
			return "", fmt.Errorf("unexpected command %q", command)
		}
	}

	updated := enrichWithRunner(response, runner)
	if !containsPrefix(updated.Findings, "Kubernetes event evidence: The pasted Kubernetes event output contains failure signals") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	for _, verified := range updated.VerifiedFrom {
		if strings.Contains(verified, "kubectl logs") {
			t.Fatalf("VerifiedFrom should not include log probes for image pull failures: %#v", updated.VerifiedFrom)
		}
	}
	combined := strings.Join(updated.NextSteps, " ")
	if !strings.Contains(combined, "imagePullSecrets") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if !strings.Contains(combined, "dig +short ghcr.io") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
