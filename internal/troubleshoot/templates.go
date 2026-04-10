package troubleshoot

import (
	"strings"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

func Resolve(query string, collector evidence.Collector) (models.Response, bool) {
	normalized := strings.ToLower(strings.TrimSpace(query))
	switch {
	case strings.Contains(normalized, "crashloopbackoff"), strings.Contains(normalized, "crash loop back off"):
		return k8sCrashLoop(collector), true
	case strings.Contains(normalized, "imagepullbackoff"), strings.Contains(normalized, "errimagepull"):
		return k8sImagePullBackoff(collector), true
	case strings.Contains(normalized, "pod pending"), strings.Contains(normalized, "pending pod"), strings.Contains(normalized, "pods are pending"):
		return k8sPendingPod(collector), true
	case strings.Contains(normalized, "container unhealthy"), strings.Contains(normalized, "unhealthy container"):
		return runtimeUnhealthy(query, collector), true
	case strings.Contains(normalized, "image pull failed"), strings.Contains(normalized, "failed to pull image"), strings.Contains(normalized, "imagepullbackoff"):
		return runtimeImagePull(query, collector), true
	case (strings.Contains(normalized, "container") || strings.Contains(normalized, "docker") || strings.Contains(normalized, "podman")) &&
		(strings.Contains(normalized, "not starting") || strings.Contains(normalized, "failed to start")):
		return runtimeStartFailure(query, collector), true
	case strings.Contains(normalized, "not starting"), strings.Contains(normalized, "failed to start"):
		return serviceFailure(query, collector), true
	case strings.Contains(normalized, "connection refused"), strings.Contains(normalized, "can't connect"), strings.Contains(normalized, "cannot connect"):
		return networkConnect(query, collector), true
	case strings.Contains(normalized, "no space left"), strings.Contains(normalized, "disk full"):
		return diskFull(collector), true
	default:
		return models.Response{}, false
	}
}

func k8sCrashLoop(collector evidence.Collector) models.Response {
	command := "kubectl get pods -A"
	ev := collector.Lookup("kubectl")
	response := models.Response{
		IntentID:       "troubleshoot_k8s_crashloopbackoff",
		Command:        command,
		Explanation:    "Start by identifying which pod is crashing, then inspect logs and describe output before changing the workload.",
		ExpectedOutput: "A pod table showing CrashLoopBackOff status and restart counts.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps: []string{
			"Run `kubectl logs <pod> --tail=100` to inspect the failing container output.",
			"Run `kubectl describe pod <pod>` to review events, restart reasons, and probe failures.",
			"Check config maps, secrets, and environment variables referenced by the crashing workload before any rollout action.",
		},
	}
	addHelpEvidence(&response, ev, "kubectl")
	return response
}

func k8sImagePullBackoff(collector evidence.Collector) models.Response {
	command := "kubectl get events -A --sort-by=.metadata.creationTimestamp"
	ev := collector.Lookup("kubectl")
	response := models.Response{
		IntentID:       "troubleshoot_k8s_imagepullbackoff",
		Command:        command,
		Explanation:    "Check recent cluster events first to confirm whether the issue is image naming, registry auth, DNS, or network reachability.",
		ExpectedOutput: "Recent events showing image pull failures, registry auth errors, or DNS and network-related messages.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps: []string{
			"Run `kubectl describe pod <pod>` to see the exact pull error on the failing container.",
			"Verify the image name, tag, and imagePullSecrets referenced by the workload.",
			"Check node or runtime-level registry connectivity only after confirming the workload spec is correct.",
		},
	}
	addHelpEvidence(&response, ev, "kubectl")
	return response
}

func k8sPendingPod(collector evidence.Collector) models.Response {
	command := "kubectl get pods -A"
	ev := collector.Lookup("kubectl")
	response := models.Response{
		IntentID:       "troubleshoot_k8s_pending_pod",
		Command:        command,
		Explanation:    "Start by identifying pending pods, then inspect scheduler events and resource constraints before changing replicas or node state.",
		ExpectedOutput: "A pod table showing Pending status for unscheduled or blocked workloads.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps: []string{
			"Run `kubectl describe pod <pod>` to inspect scheduler messages and volume-related errors.",
			"Check whether the cluster has enough CPU, memory, or matching node labels and taints for the workload.",
			"Review PVC binding and storage class events if the pod waits on storage.",
		},
	}
	addHelpEvidence(&response, ev, "kubectl")
	return response
}

func runtimeUnhealthy(query string, collector evidence.Collector) models.Response {
	runtime := runtimeFromQuery(query)
	command := runtime + " ps -a"
	ev := collector.Lookup(runtime)
	response := models.Response{
		IntentID:       "troubleshoot_runtime_unhealthy",
		Command:        command,
		Explanation:    "Start by confirming container status and health output before restarting or recreating anything.",
		ExpectedOutput: "Container status lines, including health or exit status when the runtime reports them.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps: []string{
			"Run `" + runtime + " inspect <container>` to review health-check configuration and restart policy.",
			"Run `" + runtime + " logs --tail 100 <container>` to inspect recent application errors.",
			"If the container exits quickly, compare the image entrypoint and required environment variables before restarting it.",
		},
	}
	addHelpEvidence(&response, ev, runtime)
	return response
}

func runtimeStartFailure(query string, collector evidence.Collector) models.Response {
	runtime := runtimeFromQuery(query)
	command := runtime + " ps -a"
	ev := collector.Lookup(runtime)
	response := models.Response{
		IntentID:       "troubleshoot_runtime_start_failure",
		Command:        command,
		Explanation:    "Check the container state first, then inspect logs and low-level metadata before any restart or recreate action.",
		ExpectedOutput: "Container state, exit codes, and recent status strings that show whether the container is restarting, exited, or never launched correctly.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps: []string{
			"Run `" + runtime + " logs --tail 100 <container>` for startup errors from the application process.",
			"Run `" + runtime + " inspect <container>` to review mounts, image, command, and restart policy.",
			"If the runtime itself looks unhealthy, run `systemctl status " + runtime + " --no-pager -l`.",
		},
	}
	addHelpEvidence(&response, ev, runtime)
	return response
}

func runtimeImagePull(query string, collector evidence.Collector) models.Response {
	runtime := runtimeFromQuery(query)
	command := runtime + " images"
	ev := collector.Lookup(runtime)
	response := models.Response{
		IntentID:       "troubleshoot_runtime_image_pull",
		Command:        command,
		Explanation:    "Start by confirming whether the image is already cached locally, then check runtime and registry-facing errors before retrying a pull.",
		ExpectedOutput: "A list of locally available images. If the requested image is absent, the next checks should focus on naming, auth, and registry connectivity.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps: []string{
			"Verify the image name and tag exactly match the registry artifact you expect.",
			"Run `" + runtime + " login <registry>` only if authentication is required and credentials are available.",
			"Check runtime logs or service status if pull failures mention TLS, DNS, or registry connectivity issues.",
		},
	}
	addHelpEvidence(&response, ev, runtime)
	return response
}

func runtimeFromQuery(query string) string {
	lower := strings.ToLower(query)
	switch {
	case strings.Contains(lower, "podman"):
		return "podman"
	case strings.Contains(lower, "containerd"):
		if strings.Contains(lower, "nerdctl") {
			return "nerdctl"
		}
		if strings.Contains(lower, "crictl") {
			return "crictl"
		}
		if strings.Contains(lower, "ctr") {
			return "ctr"
		}
		return "containerd"
	default:
		return "docker"
	}
}

func serviceFailure(query string, collector evidence.Collector) models.Response {
	service := "service"
	fields := strings.Fields(strings.ToLower(query))
	if len(fields) > 0 {
		service = fields[0]
	}

	command := "systemctl status " + service + " --no-pager -l"
	ev := collector.Lookup("systemctl")
	response := models.Response{
		IntentID:       "troubleshoot_service_failure",
		Command:        command,
		Explanation:    "Start with service state and recent logs before attempting any restart. This keeps troubleshooting read-only and surfaces the immediate failure reason first.",
		ExpectedOutput: "The unit state, recent log lines, exit codes, and clues about why the service is failing to start.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps: []string{
			"Run `journalctl -u " + service + " -n 50 --no-pager` to inspect the most recent service logs.",
			"Look for missing config files, permission errors, or port conflicts before trying a restart.",
			"If logs mention a port conflict, run `ss -ltnp` and check whether another process is already bound.",
		},
	}
	addHelpEvidence(&response, ev, "systemctl")
	return response
}

func networkConnect(query string, collector evidence.Collector) models.Response {
	command := "ss -ltnp"
	ev := collector.Lookup("ss")
	response := models.Response{
		IntentID:       "troubleshoot_network_connectivity",
		Command:        command,
		Explanation:    "Start by checking whether the expected service is actually listening locally, then verify host reachability and HTTP behavior as separate steps.",
		ExpectedOutput: "A list of listening TCP sockets. If the expected service is absent, the issue is likely local service startup rather than routing.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps: []string{
			"Run `ip addr show` to confirm the expected interface and address are present.",
			"Run `ping -c 4 <host>` for basic reachability when ICMP is allowed.",
			"For HTTP services, run `curl -I <url>` to confirm status codes and basic response headers.",
		},
	}
	addHelpEvidence(&response, ev, "ss")
	return response
}

func diskFull(collector evidence.Collector) models.Response {
	command := "df -h"
	ev := collector.Lookup("df")
	response := models.Response{
		IntentID:       "troubleshoot_disk_full",
		Command:        command,
		Explanation:    "Check filesystem capacity first, then narrow the largest directories before considering cleanup actions. This keeps the workflow safe and evidence-driven.",
		ExpectedOutput: "Filesystem totals and free space. A nearly full mount point identifies where to focus deeper inspection.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps: []string{
			"Run `du -sh /var/* 2>/dev/null | sort -h` or the equivalent for the full filesystem on the affected mount.",
			"Run `find /var -type f -size +1G` to identify unusually large files when `/var` is involved.",
			"Prefer log rotation or targeted cleanup over broad deletion commands.",
		},
	}
	addHelpEvidence(&response, ev, "df")
	return response
}

func confidenceFor(ev evidence.Evidence) string {
	switch {
	case ev.Exists && len(ev.HelpSnippet) > 0:
		return "High"
	case ev.Exists:
		return "Medium"
	default:
		return "Low"
	}
}

func addHelpEvidence(response *models.Response, ev evidence.Evidence, command string) {
	if len(ev.HelpSnippet) > 0 {
		response.VerifiedFrom = append(response.VerifiedFrom, command+" --help")
	}
	if !ev.Exists {
		response.Warnings = append(response.Warnings, command+" is not currently installed on this host")
	}
}
