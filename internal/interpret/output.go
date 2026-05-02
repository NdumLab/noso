package interpret

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

var dependencyHostPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)connection to server at\s+([a-z0-9][a-z0-9._-]*)`),
	regexp.MustCompile(`(?i)lookup\s+([a-z0-9][a-z0-9._-]*)\s+on`),
	regexp.MustCompile(`(?i)(?:dial tcp|connect)\s+([a-z0-9][a-z0-9._-]*)[: ]`),
	regexp.MustCompile(`(?i)(?:host|hostname|server)\s+([a-z0-9][a-z0-9._-]*)`),
}

var dependencyPortPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bport\s+([0-9]{2,5})\b`),
	regexp.MustCompile(`(?i)[a-z0-9][a-z0-9._-]*:([0-9]{2,5})\b`),
}

var imageReferencePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)image\s+"([^"]+)"`),
	regexp.MustCompile(`(?i)\bpulling image\s+"([^"]+)"`),
	regexp.MustCompile(`(?i)\bfrom\s+([a-z0-9][a-z0-9._:-]*/[a-z0-9._/-]+(?::[a-z0-9._-]+)?)`),
}

var kubernetesContainerHintPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)defaulted container\s+"?([a-z0-9][a-z0-9._-]*)"?\s+out of:`),
	regexp.MustCompile(`(?i)failed container\s+"?([a-z0-9][a-z0-9._-]*)"?`),
	regexp.MustCompile(`(?i)container\s+"?([a-z0-9][a-z0-9._-]*)"?\s+(?:in pod|is|was)`),
	regexp.MustCompile(`(?i)restarting failed container\s+"?([a-z0-9][a-z0-9._-]*)"?`),
}

func Output(command string, text string) (models.Response, error) {
	command = strings.TrimSpace(command)
	text = strings.TrimSpace(text)
	if command == "" {
		return models.Response{}, fmt.Errorf("command is required")
	}

	lower := strings.ToLower(command)
	switch {
	case strings.HasPrefix(lower, "systemctl status"):
		return interpretSystemctlStatus(command, text), nil
	case strings.HasPrefix(lower, "docker ps"), strings.HasPrefix(lower, "podman ps"):
		return interpretRuntimePS(command, text), nil
	case strings.HasPrefix(lower, "docker logs"), strings.HasPrefix(lower, "podman logs"):
		return interpretRuntimeLogs(command, text), nil
	case strings.HasPrefix(lower, "df "):
		return interpretDF(command, text), nil
	case strings.HasPrefix(lower, "free "):
		return interpretFree(command, text), nil
	case strings.HasPrefix(lower, "kubectl get pods"):
		return interpretKubectlGetPods(command, text), nil
	case strings.HasPrefix(lower, "kubectl get events"):
		return interpretKubectlGetEvents(command, text), nil
	case strings.HasPrefix(lower, "kubectl describe pod"):
		return interpretKubectlDescribePod(command, text), nil
	case strings.HasPrefix(lower, "kubectl logs"):
		return interpretKubectlLogs(command, text), nil
	case strings.HasPrefix(lower, "journalctl"):
		return interpretJournalctl(command, text), nil
	case strings.HasPrefix(lower, "ps "):
		return interpretPS(command, text), nil
	case strings.HasPrefix(lower, "ip addr"):
		return interpretIPAddr(command, text), nil
	default:
		return models.Response{
			IntentID:       "interpret_unsupported_output",
			Command:        command,
			Explanation:    "This command output does not have a dedicated interpreter yet. The pasted output was accepted, but only limited generic interpretation is available.",
			ExpectedOutput: "A concise explanation of pasted command output once support is added for this command.",
			Risk:           safety.RiskLow,
			Confidence:     "Low",
			Warnings:       []string{"no dedicated interpreter exists yet for this command"},
		}, nil
	}
}

func interpretSystemctlStatus(command string, text string) models.Response {
	lower := strings.ToLower(text)
	response := baseInterpretResponse(command)
	response.IntentID = "interpret_systemctl_status"
	response.ExpectedOutput = "A summary of unit health, failure signals, and the next safe diagnostics to run."

	switch {
	case strings.Contains(lower, "could not be found"), strings.Contains(lower, "loaded: not-found"):
		response.Explanation = "The requested unit could not be found on this host. That usually means the service name is wrong, the unit file is absent, or the workload is not managed by systemd."
		response.Confidence = "High"
		response.NextSteps = []string{
			"Confirm the service name with `systemctl list-units --type=service | grep <name>`.",
			"If the workload is containerized, inspect the runtime instead of systemd.",
		}
	case strings.Contains(lower, "active: failed"):
		response.Explanation = "The unit is in a failed state. systemd recorded a service failure rather than a healthy running process."
		response.Confidence = "High"
		response.NextSteps = []string{
			"Run `journalctl -u <service> -n 50 --no-pager` to inspect the most recent failure logs.",
			"Run `systemctl show <service> -p ExecMainStatus -p Result` to confirm the exit status and result reason.",
		}
	case strings.Contains(lower, "active: active (running)"):
		response.Explanation = "The unit appears healthy and running. The pasted status output does not indicate a current service failure."
		response.Confidence = "High"
		response.NextSteps = []string{
			"Inspect recent logs with `journalctl -u <service> -n 50 --no-pager` if the application still behaves incorrectly.",
		}
	case strings.Contains(lower, "active: inactive (dead)"):
		response.Explanation = "The unit is inactive and not currently running. That may be expected for one-shot units, but it usually means the service is stopped."
		response.Confidence = "High"
		response.NextSteps = []string{
			"Check whether the unit should normally stay active with `systemctl cat <service>`.",
			"Inspect the prior logs with `journalctl -u <service> -n 50 --no-pager`.",
		}
	default:
		response.Explanation = "The pasted systemctl output was recognized, but the current parser could not confidently classify the unit as running, failed, or inactive."
		response.Confidence = "Medium"
		response.NextSteps = []string{
			"Look for the `Active:` and `Result:` lines in the pasted status output.",
			"Use `journalctl -u <service> -n 50 --no-pager` for more detail if the unit is unhealthy.",
		}
	}

	if strings.Contains(lower, "result: exit-code") {
		response.NextSteps = appendUnique(response.NextSteps, "The unit exited with a non-zero code; inspect the command or startup script that systemd launched.")
	}
	return response
}

func interpretDF(command string, text string) models.Response {
	response := baseInterpretResponse(command)
	response.IntentID = "interpret_df"
	response.ExpectedOutput = "A summary of filesystem capacity pressure and any mounts that are close to full."

	var hot []string
	var parsed int
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Filesystem") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		parsed++
		use := strings.TrimSuffix(fields[4], "%")
		pct, err := strconv.Atoi(use)
		if err != nil {
			continue
		}
		if pct >= 90 {
			hot = append(hot, fmt.Sprintf("%s at %d%% on %s", fields[0], pct, fields[len(fields)-1]))
		}
	}

	switch {
	case parsed == 0:
		response.Explanation = "The pasted output did not look like standard `df` output, so no filesystem-capacity interpretation was possible."
		response.Confidence = "Low"
		response.Warnings = []string{"unable to parse filesystem rows from provided output"}
	case len(hot) > 0:
		response.Explanation = "At least one filesystem is close to full. Capacity pressure is likely to affect writes, package installs, logs, or service startup on those mounts."
		response.Confidence = "High"
		for _, item := range hot {
			response.NextSteps = append(response.NextSteps, "Investigate "+item)
		}
		response.NextSteps = append(response.NextSteps,
			"Run `du -sh <mount>/* | sort -h` on the affected mount to find large directories.",
		)
	default:
		response.Explanation = "No parsed filesystem in the pasted output is above the 90% usage threshold, so there is no obvious critical disk-pressure signal."
		response.Confidence = "High"
	}
	return response
}

func interpretFree(command string, text string) models.Response {
	response := baseInterpretResponse(command)
	response.IntentID = "interpret_free"
	response.ExpectedOutput = "A summary of memory pressure based on total versus available memory and swap usage."

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Mem:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 7 {
			break
		}
		total, okTotal := parseHumanBytes(fields[1])
		available, okAvail := parseHumanBytes(fields[6])
		if !okTotal || !okAvail || total == 0 {
			break
		}
		availablePct := (float64(available) / float64(total)) * 100
		switch {
		case availablePct < 10:
			response.Explanation = fmt.Sprintf("The host appears to be under memory pressure. Only about %.0f%% of memory is currently available.", math.Round(availablePct))
			response.Confidence = "High"
			response.NextSteps = []string{
				"Run `ps aux --sort=-%mem | head` to identify memory-heavy processes.",
				"Compare memory pressure with `journalctl -k | tail` for OOM killer activity.",
			}
		case availablePct < 20:
			response.Explanation = fmt.Sprintf("Available memory is reduced at about %.0f%% of total. The host is not necessarily critical, but memory pressure is worth watching.", math.Round(availablePct))
			response.Confidence = "High"
			response.NextSteps = []string{
				"Run `ps aux --sort=-%mem | head` to confirm which processes are consuming the most memory.",
			}
		default:
			response.Explanation = fmt.Sprintf("The pasted memory output shows about %.0f%% of memory still available, so there is no obvious severe memory-pressure signal.", math.Round(availablePct))
			response.Confidence = "High"
		}
		return response
	}

	response.Explanation = "The pasted output did not look like standard `free` output, so memory pressure could not be interpreted confidently."
	response.Confidence = "Low"
	response.Warnings = []string{"unable to parse memory values from provided output"}
	return response
}

func interpretKubectlGetPods(command string, text string) models.Response {
	response := baseInterpretResponse(command)
	response.IntentID = "interpret_kubectl_get_pods"
	response.ExpectedOutput = "A summary of pod health states and the most likely next Kubernetes diagnostics."

	var issues []string
	var parsed int
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if line == "" || strings.HasPrefix(line, "NAME ") || strings.HasPrefix(lower, "namespace ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		parsed++
		name := fields[0]
		status := fields[2]
		if len(fields) >= 5 {
			name = fields[1]
			status = fields[3]
		}
		if status != "Running" && status != "Completed" {
			issues = append(issues, fmt.Sprintf("%s is in %s", name, status))
		}
	}

	switch {
	case parsed == 0:
		response.Explanation = "The pasted output did not look like standard `kubectl get pods` output, so pod-state interpretation was limited."
		response.Confidence = "Low"
		response.Warnings = []string{"unable to parse pod rows from provided output"}
	case len(issues) > 0:
		response.Explanation = "At least one pod is not healthy or not fully running. The cluster likely needs pod-level inspection rather than assuming the workload is healthy."
		response.Confidence = "High"
		for _, issue := range issues {
			response.NextSteps = append(response.NextSteps, "Inspect "+issue)
		}
		response.NextSteps = append(response.NextSteps,
			"Run `kubectl describe pod <pod> -n <namespace>` to inspect events and scheduling details.",
			"Run `kubectl logs <pod> -n <namespace> --previous` if the pod is restarting.",
		)
	default:
		response.Explanation = "All parsed pods are either Running or Completed, so the pasted output does not show an obvious unhealthy pod state."
		response.Confidence = "High"
	}
	return response
}

func interpretKubectlGetEvents(command string, text string) models.Response {
	response := baseInterpretResponse(command)
	response.IntentID = "interpret_kubectl_get_events"
	response.ExpectedOutput = "A summary of recent Kubernetes failure events, including affected pod names and likely next diagnostics."
	response.ContainerHint = extractKubernetesContainerHint(command, text)

	lower := strings.ToLower(text)
	hasRestartBackoff := strings.Contains(lower, "back-off restarting failed container") || strings.Contains(lower, "crashloopbackoff")
	hasImagePull := strings.Contains(lower, "imagepullbackoff") || strings.Contains(lower, "errimagepull") || strings.Contains(lower, "failed to pull image")
	hasScheduling := strings.Contains(lower, "failedscheduling")
	hasMountFailure := strings.Contains(lower, "failedmount")
	var signals []string
	for _, marker := range []struct {
		needle string
		label  string
	}{
		{"back-off restarting failed container", "restart backoff detected"},
		{"crashloopbackoff", "CrashLoopBackOff event detected"},
		{"imagepullbackoff", "ImagePullBackOff event detected"},
		{"errimagepull", "ErrImagePull event detected"},
		{"failed to pull image", "image pull failure detected"},
		{"failedscheduling", "scheduler failure detected"},
		{"failedmount", "volume mount failure detected"},
		{"oomkilled", "OOMKilled event detected"},
	} {
		if strings.Contains(lower, marker.needle) {
			signals = append(signals, marker.label)
		}
	}

	pod := ExtractEventPodForTroubleshoot(text)
	namespace := ExtractEventNamespaceForTroubleshoot(command, text)
	deployment := extractKubernetesObjectReference(text, "deployment")
	service := extractKubernetesObjectReference(text, "service")

	if len(signals) == 0 {
		response.Explanation = "The pasted event output was recognized, but no common failure signals were detected in the current rows."
		response.Confidence = "Medium"
		response.NextSteps = []string{
			"Inspect the newest Warning rows for involved objects and controller messages.",
		}
		if pod != "" {
			describe := "kubectl describe pod " + pod
			if namespace != "" {
				describe = "kubectl describe pod -n " + namespace + " " + pod
			}
			response.NextSteps = append(response.NextSteps, "Run `"+describe+"` to inspect the affected pod in more detail.")
		}
		return response
	}

	response.Explanation = "The pasted Kubernetes event output contains failure signals: " + strings.Join(signals, "; ") + "."
	response.Confidence = "High"
	if pod != "" {
		describe := "kubectl describe pod " + pod
		logs := "kubectl logs " + pod + " --previous"
		if namespace != "" {
			describe = "kubectl describe pod -n " + namespace + " " + pod
			logs = "kubectl logs -n " + namespace + " " + pod + " --previous"
		}
		if response.ContainerHint != "" {
			if namespace != "" {
				logs = "kubectl logs -n " + namespace + " " + pod + " -c " + response.ContainerHint + " --previous"
			} else {
				logs = "kubectl logs " + pod + " -c " + response.ContainerHint + " --previous"
			}
		}
		response.NextSteps = append(response.NextSteps, "Run `"+describe+"` to inspect pod conditions and related events in detail.")
		switch {
		case hasRestartBackoff:
			response.NextSteps = append(response.NextSteps, "Run `"+logs+"` to inspect the last failing container exit.")
		case hasImagePull:
			registryHost, registryPort := extractImageRegistryEndpoint(text)
			response.NextSteps = append(response.NextSteps,
				"Verify the image name, tag, and imagePullSecrets referenced by the workload before retrying any rollout.",
				"Check node-to-registry DNS and network reachability only after the workload spec and credentials look correct.",
			)
			if registryHost != "" {
				response.NextSteps = append(response.NextSteps,
					"Run `dig +short "+registryHost+"` or `nslookup "+registryHost+"` to confirm DNS resolution for the referenced image registry.",
				)
			}
			if registryHost != "" && registryPort != "" {
				response.NextSteps = append(response.NextSteps,
					"Run `nc -vz "+registryHost+" "+registryPort+"` to verify the registry listener is reachable on the expected port.",
				)
			}
		case hasScheduling:
			response.NextSteps = append(response.NextSteps, schedulerNextSteps(text)...)
		case hasMountFailure:
			response.NextSteps = append(response.NextSteps, mountFailureNextSteps(text, namespace)...)
		default:
			response.NextSteps = append(response.NextSteps, "Run `"+logs+"` to inspect the last failing container exit once the workload has started at least once.")
		}
	} else {
		response.NextSteps = append(response.NextSteps,
			"Identify the affected pod from the Involved Object or Message columns, then inspect it with `kubectl describe pod <pod>`.",
		)
		switch {
		case hasRestartBackoff:
			response.NextSteps = append(response.NextSteps, "Run `kubectl logs <pod> --previous` once the failing pod is confirmed.")
		case hasImagePull:
			registryHost, registryPort := extractImageRegistryEndpoint(text)
			response.NextSteps = append(response.NextSteps,
				"Verify the image name, tag, and imagePullSecrets referenced by the affected workload.",
				"Check registry DNS and network reachability from the relevant node or pod path only after the workload spec is confirmed.",
			)
			if registryHost != "" {
				response.NextSteps = append(response.NextSteps,
					"Run `dig +short "+registryHost+"` or `nslookup "+registryHost+"` to confirm DNS resolution for the referenced image registry.",
				)
			}
			if registryHost != "" && registryPort != "" {
				response.NextSteps = append(response.NextSteps,
					"Run `nc -vz "+registryHost+" "+registryPort+"` to verify the registry listener is reachable on the expected port.",
				)
			}
		case hasScheduling:
			response.NextSteps = append(response.NextSteps, schedulerNextSteps(text)...)
		case hasMountFailure:
			response.NextSteps = append(response.NextSteps, mountFailureNextSteps(text, namespace)...)
		default:
			response.NextSteps = append(response.NextSteps, "Run `kubectl logs <pod> --previous` once the failing pod is confirmed.")
		}
	}
	response.NextSteps = append(response.NextSteps, eventOwnerObjectNextSteps(namespace, deployment, service)...)
	return response
}

func interpretKubectlDescribePod(command string, text string) models.Response {
	response := baseInterpretResponse(command)
	response.IntentID = "interpret_kubectl_describe_pod"
	response.ExpectedOutput = "A summary of pod state, container failure signals, and the next safe Kubernetes diagnostics."
	response.ContainerHint = extractKubernetesContainerHint(command, text)

	lower := strings.ToLower(text)
	var signals []string
	for _, marker := range []struct {
		needle string
		label  string
	}{
		{"crashloopbackoff", "CrashLoopBackOff detected"},
		{"imagepullbackoff", "ImagePullBackOff detected"},
		{"errimagepull", "ErrImagePull detected"},
		{"oomkilled", "OOMKilled container state detected"},
		{"failedmount", "volume mount failure detected"},
		{"failedscheduling", "scheduler failure detected"},
		{"reason: pending", "pending pod state detected"},
	} {
		if strings.Contains(lower, marker.needle) {
			signals = append(signals, marker.label)
		}
	}

	if len(signals) == 0 {
		response.Explanation = "The pasted pod description was recognized, but no common failure markers were detected in the current text."
		response.Confidence = "Medium"
		response.NextSteps = []string{
			"Inspect the Events section for scheduler, image-pull, or mount failures.",
			"Run `kubectl logs <pod> --tail=100` if the pod starts and emits container logs.",
		}
		return response
	}

	response.Explanation = "The pasted pod description contains failure signals: " + strings.Join(signals, "; ") + "."
	response.Confidence = "High"
	response.NextSteps = []string{
		"Review the Events section around the first failure signal.",
		"Run `kubectl logs <pod> --tail=100` for container-level errors if the pod started at least once.",
	}
	return response
}

func interpretKubectlLogs(command string, text string) models.Response {
	response := interpretGenericLogs(command, text)
	response.IntentID = "interpret_kubectl_logs"
	response.ExpectedOutput = "A summary of application-level error signals from pod logs."
	response.ContainerHint = extractKubernetesContainerHint(command, text)
	return response
}

func interpretRuntimePS(command string, text string) models.Response {
	response := baseInterpretResponse(command)
	response.IntentID = "interpret_runtime_ps"
	response.ExpectedOutput = "A summary of container runtime state, especially exited, restarting, or unhealthy containers."

	lower := strings.ToLower(text)
	var unhealthy []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(strings.ToLower(trimmed), "container id") || strings.HasPrefix(strings.ToLower(trimmed), "con_id") {
			continue
		}
		if strings.Contains(strings.ToLower(trimmed), "exited") ||
			strings.Contains(strings.ToLower(trimmed), "restarting") ||
			strings.Contains(strings.ToLower(trimmed), "created") ||
			strings.Contains(strings.ToLower(trimmed), "unhealthy") {
			unhealthy = append(unhealthy, trimmed)
		}
	}

	switch {
	case len(unhealthy) > 0:
		response.Explanation = "The runtime container list shows non-running or unhealthy containers."
		response.Confidence = "High"
		response.NextSteps = []string{
			"Identify the affected container row and inspect recent logs for the same container.",
			"Review restart counts, exit status, and image configuration before restarting anything.",
		}
	case strings.Contains(lower, "up "):
		response.Explanation = "The runtime container list shows running containers and no obvious exited or restarting states in the pasted output."
		response.Confidence = "Medium"
	default:
		response.Explanation = "The pasted runtime output was recognized, but no clear container state could be classified from it."
		response.Confidence = "Low"
	}
	return response
}

func interpretRuntimeLogs(command string, text string) models.Response {
	response := interpretGenericLogs(command, text)
	response.IntentID = "interpret_runtime_logs"
	response.ExpectedOutput = "A summary of application-level error signals from container logs."
	return response
}

func baseInterpretResponse(command string) models.Response {
	return models.Response{
		Command:    command,
		Risk:       safety.RiskLow,
		Confidence: "Medium",
	}
}

func parseHumanBytes(value string) (uint64, bool) {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return 0, false
	}
	value = strings.TrimSuffix(value, "B")
	value = strings.TrimSuffix(value, "I")

	multiplier := float64(1)
	last := value[len(value)-1]
	switch last {
	case 'K':
		multiplier = 1024
		value = value[:len(value)-1]
	case 'M':
		multiplier = 1024 * 1024
		value = value[:len(value)-1]
	case 'G':
		multiplier = 1024 * 1024 * 1024
		value = value[:len(value)-1]
	case 'T':
		multiplier = 1024 * 1024 * 1024 * 1024
		value = value[:len(value)-1]
	}

	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return uint64(number * multiplier), true
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func interpretJournalctl(command string, text string) models.Response {
	response := interpretGenericLogs(command, text)
	response.IntentID = "interpret_journalctl"
	response.ExpectedOutput = "A summary of error signals, failure patterns, and the next safe diagnostics."
	return response
}

func interpretGenericLogs(command string, text string) models.Response {
	response := baseInterpretResponse(command)
	lower := strings.ToLower(text)
	var signals []string

	if strings.Contains(lower, "failed") || strings.Contains(lower, "error") {
		signals = append(signals, "error or failure lines detected")
	}
	if strings.Contains(lower, "oom") || strings.Contains(lower, "out of memory") || strings.Contains(lower, "killed process") {
		signals = append(signals, "OOM killer activity detected")
	}
	if strings.Contains(lower, "segfault") || strings.Contains(lower, "segmentation fault") {
		signals = append(signals, "segmentation fault detected")
	}
	if strings.Contains(lower, "permission denied") {
		signals = append(signals, "permission denied errors detected")
	}
	if strings.Contains(lower, "connection refused") || strings.Contains(lower, "connection timed out") {
		signals = append(signals, "network connection errors detected")
	}
	if isDatabaseConnectivityLog(lower, text) {
		signals = append(signals, "database connectivity errors detected")
	}
	if strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "name or service not known") ||
		strings.Contains(lower, "temporary failure in name resolution") ||
		strings.Contains(lower, "server misbehaving") {
		signals = append(signals, "dns resolution errors detected")
	}
	if strings.Contains(lower, "no space left") {
		signals = append(signals, "disk-full errors detected")
	}

	if len(signals) == 0 {
		if strings.TrimSpace(text) == "" {
			response.Explanation = "The pasted log output appears empty. The process may have no recent entries yet, or the selected window returned no records."
			response.Confidence = "Low"
		} else {
			response.Explanation = "No obvious failure signals were found in the pasted log output. The lines do not contain common error keywords."
			response.Confidence = "Medium"
			response.NextSteps = []string{
				"Scan the output manually for unusual state transitions or unexpected restarts.",
				"Widen the log window or fetch more lines if the failure is intermittent.",
			}
		}
		return response
	}

	response.Explanation = fmt.Sprintf("The pasted log output contains %d signal(s): %s.",
		len(signals), strings.Join(signals, "; "))
	response.Confidence = "High"
	response.NextSteps = defaultLogContextNextSteps(command)
	if strings.Contains(lower, "oom") || strings.Contains(lower, "killed process") {
		response.NextSteps = append(response.NextSteps,
			"Run `free -h` to inspect current memory state.",
			"Run `ps aux --sort=-%mem | head` to identify memory-heavy processes.")
	}
	if isDatabaseConnectivityLog(lower, text) {
		host := extractDependencyHost(text)
		port := extractDependencyPort(text)
		if host == "" {
			host = "<database-host>"
		}
		response.NextSteps = append(response.NextSteps,
			"Run `dig +short "+host+"` or `nslookup "+host+"` to confirm DNS resolution for the configured database endpoint.",
		)
		if port != "" {
			response.NextSteps = append(response.NextSteps,
				"Run `nc -vz "+host+" "+port+"` to verify the upstream listener is reachable on the expected database port.",
			)
		} else {
			response.NextSteps = append(response.NextSteps,
				"Run `ss -ltnp` on the database host or through the expected access path to confirm the listener is up on the required port.",
			)
		}
	}
	if strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "name or service not known") ||
		strings.Contains(lower, "temporary failure in name resolution") ||
		strings.Contains(lower, "server misbehaving") {
		host := extractDependencyHost(text)
		if host == "" {
			host = "<hostname>"
		}
		response.NextSteps = append(response.NextSteps,
			"Run `dig +short "+host+"` or `nslookup "+host+"` from the same host or container context to confirm name resolution.",
			"Inspect `/etc/resolv.conf` or the pod DNS policy if the name should resolve internally.",
		)
	}
	return response
}

func defaultLogContextNextSteps(command string) []string {
	lower := strings.ToLower(strings.TrimSpace(command))
	steps := []string{
		"Check the lines around each error for the root cause.",
	}
	switch {
	case strings.HasPrefix(lower, "kubectl logs"):
		if namespace := extractNamespaceFromKubectlCommand(command); namespace != "" {
			return append(steps, "Run `kubectl get events -n "+namespace+" --sort-by=.metadata.creationTimestamp` to inspect recent namespace-scoped scheduler, image-pull, and restart events.")
		}
		return append(steps, "Run `kubectl get events -A --sort-by=.metadata.creationTimestamp` to inspect recent cluster events related to the workload.")
	case strings.HasPrefix(lower, "docker logs"), strings.HasPrefix(lower, "podman logs"):
		runtime := "container"
		fields := strings.Fields(command)
		if len(fields) > 0 {
			runtime = fields[0]
		}
		return append(steps, "Run `"+runtime+" ps -a` to correlate the log failure with restart state and exit codes.")
	default:
		return append(steps, "Run `journalctl -u <service> -n 100 --no-pager` for more context.")
	}
}

func isDatabaseConnectivityLog(lower, text string) bool {
	if strings.Contains(lower, "failed to connect to database") ||
		strings.Contains(lower, "connect to database") ||
		strings.Contains(lower, "connection to server at") ||
		strings.Contains(lower, "sqlstate") ||
		strings.Contains(lower, "pq: ") {
		return true
	}
	if !(strings.Contains(lower, "connection refused") || strings.Contains(lower, "connection timed out")) {
		return false
	}
	host := extractDependencyHost(text)
	port := extractDependencyPort(text)
	return looksLikeDatabaseHost(host) || looksLikeDatabasePort(port)
}

func looksLikeDatabaseHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	for _, marker := range []string{"db", "postgres", "mysql", "mariadb", "mongo", "redis", "sql"} {
		if strings.Contains(host, marker) {
			return true
		}
	}
	return false
}

func looksLikeDatabasePort(port string) bool {
	switch strings.TrimSpace(port) {
	case "1433", "1521", "27017", "3306", "5432", "5433", "6379":
		return true
	default:
		return false
	}
}

func extractNamespaceFromKubectlCommand(command string) string {
	fields := strings.Fields(strings.TrimSpace(command))
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == "-n" && isKubernetesContainerName(fields[i+1]) {
			return fields[i+1]
		}
	}
	return ""
}

func extractDependencyHost(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	for _, pattern := range dependencyHostPatterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) < 2 {
			continue
		}
		candidate := sanitizeDependencyHost(matches[1])
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func extractDependencyPort(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	for _, pattern := range dependencyPortPatterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) < 2 {
			continue
		}
		candidate := sanitizeDependencyPort(matches[1])
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func extractKubernetesContainerHint(command, text string) string {
	command = strings.TrimSpace(command)
	if command != "" {
		fields := strings.Fields(command)
		for i := 0; i < len(fields)-1; i++ {
			if fields[i] == "-c" && isKubernetesContainerName(fields[i+1]) {
				return fields[i+1]
			}
		}
	}

	for _, pattern := range kubernetesContainerHintPatterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) < 2 {
			continue
		}
		candidate := strings.Trim(matches[1], "\"'` ")
		if isKubernetesContainerName(candidate) {
			return candidate
		}
	}
	return ""
}

func ExtractEventPodForTroubleshoot(text string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bpod/([a-z0-9][a-z0-9._-]*)\b`),
		regexp.MustCompile(`(?i)\bin pod\s+([a-z0-9][a-z0-9._-]*)[_\s(]`),
		regexp.MustCompile(`(?i)\bon\s+pod\s+([a-z0-9][a-z0-9._-]*)\b`),
	}
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) < 2 {
			continue
		}
		if isKubernetesContainerName(matches[1]) {
			return matches[1]
		}
	}
	return ""
}

func ExtractEventNamespaceForTroubleshoot(command, text string) string {
	fields := strings.Fields(strings.TrimSpace(command))
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == "-n" && isKubernetesContainerName(fields[i+1]) {
			return fields[i+1]
		}
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bin pod\s+[a-z0-9][a-z0-9._-]*_([a-z0-9][a-z0-9._-]*)\(`),
		regexp.MustCompile(`(?i)\bpod/([a-z0-9][a-z0-9._-]*)\s+([a-z0-9][a-z0-9._-]*)\s+\d+[smhd]`),
	}
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) < 2 {
			continue
		}
		candidate := matches[len(matches)-1]
		if isKubernetesContainerName(candidate) {
			return candidate
		}
	}
	return ""
}

func extractImageRegistryEndpoint(text string) (string, string) {
	for _, pattern := range imageReferencePatterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) < 2 {
			continue
		}
		host, port := registryEndpointFromImageReference(matches[1])
		if host != "" {
			return host, port
		}
	}
	return "", ""
}

func registryEndpointFromImageReference(value string) (string, string) {
	value = strings.TrimSpace(strings.Trim(value, "\"'`"))
	if value == "" {
		return "", ""
	}
	ref := value
	if slash := strings.Index(ref, "/"); slash > 0 {
		candidate := ref[:slash]
		if strings.Contains(candidate, ".") || strings.Contains(candidate, ":") || candidate == "localhost" {
			host := candidate
			port := ""
			if colon := strings.LastIndex(candidate, ":"); colon > 0 && colon < len(candidate)-1 && !strings.Contains(candidate[colon+1:], "]") {
				port = candidate[colon+1:]
				host = candidate[:colon]
			}
			if isRegistryHost(host) {
				return host, port
			}
		}
	}
	return "", ""
}

func isRegistryHost(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func schedulerNextSteps(text string) []string {
	lower := strings.ToLower(text)
	node := extractSchedulingNode(text)
	steps := []string{
		"Inspect scheduler messages for capacity, taint, affinity, topology, and storage constraints on the affected pod.",
	}
	switch {
	case strings.Contains(lower, "insufficient memory"):
		steps = append(steps,
			"Check node allocatable memory and current requests/limits before changing replica counts or pod resources.",
		)
	case strings.Contains(lower, "insufficient cpu"):
		steps = append(steps,
			"Check node allocatable CPU and the workload's requested CPU before changing replica counts or requests.",
		)
	case strings.Contains(lower, "had taint") || strings.Contains(lower, "had untolerated taint"):
		steps = append(steps,
			"Review node taints and the workload's tolerations before moving or recreating the pod.",
		)
	case strings.Contains(lower, "node affinity") || strings.Contains(lower, "didn't match pod's node affinity"):
		steps = append(steps,
			"Review the workload's node affinity or selector rules against actual node labels before changing scheduling policy.",
		)
	case strings.Contains(lower, "pod has unbound immediate persistentvolumeclaims"):
		steps = append(steps,
			"Inspect the referenced PVC and storage class binding before changing scheduler or node settings.",
		)
	default:
		steps = append(steps,
			"Check node allocatable resources and required labels or tolerations before changing replica counts.",
		)
	}
	if node != "" {
		steps = append(steps, "Run `kubectl describe node "+node+"` to inspect the candidate node's allocatable resources, labels, taints, and recent conditions.")
	}
	return steps
}

func mountFailureNextSteps(text, namespace string) []string {
	pvc := extractQuotedValue(text, `(?i)persistentvolumeclaim\s+"([^"]+)"`)
	secret := extractQuotedValue(text, `(?i)secret\s+"([^"]+)"`)
	configMap := extractQuotedValue(text, `(?i)configmap\s+"([^"]+)"`)

	steps := []string{
		"Inspect the referenced PVC, storage class, secret, or config map in the event stream before restarting the workload.",
	}
	switch {
	case pvc != "":
		command := "kubectl describe pvc " + pvc
		if namespace != "" {
			command = "kubectl describe pvc -n " + namespace + " " + pvc
		}
		steps = append(steps,
			"Run `"+command+"` to inspect PVC `"+pvc+"` and its storage class binding before changing scheduler or node settings.",
		)
	case secret != "":
		command := "kubectl describe secret " + secret
		if namespace != "" {
			command = "kubectl describe secret -n " + namespace + " " + secret
		}
		steps = append(steps,
			"Run `"+command+"` to inspect Secret `"+secret+"` and confirm it exists in the workload namespace before recreating the pod.",
		)
	case configMap != "":
		command := "kubectl describe configmap " + configMap
		if namespace != "" {
			command = "kubectl describe configmap -n " + namespace + " " + configMap
		}
		steps = append(steps,
			"Run `"+command+"` to inspect ConfigMap `"+configMap+"` and confirm it exists in the workload namespace before recreating the pod.",
		)
	default:
		steps = append(steps,
			"Check storage class binding and kubelet mount-related events on the affected node if the volume should already exist.",
		)
	}
	return steps
}

func extractQuotedValue(text, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func extractSchedulingNode(text string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bnode/([a-z0-9][a-z0-9._-]*)\b`),
		regexp.MustCompile(`(?i)\bon node "?([a-z0-9][a-z0-9._-]*)"?`),
		regexp.MustCompile(`(?i)\bnode "?([a-z0-9][a-z0-9._-]*)"? had untolerated taint`),
		regexp.MustCompile(`(?i)\bnode "?([a-z0-9][a-z0-9._-]*)"? didn't match pod's node affinity`),
	}
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) < 2 {
			continue
		}
		candidate := strings.TrimSpace(matches[1])
		if isKubernetesContainerName(candidate) {
			return candidate
		}
	}
	return ""
}

func extractKubernetesObjectReference(text, kind string) string {
	pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(kind) + `/([a-z0-9][a-z0-9._-]*)\b`)
	matches := pattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	candidate := strings.TrimSpace(matches[1])
	if isKubernetesContainerName(candidate) {
		return candidate
	}
	return ""
}

func eventOwnerObjectNextSteps(namespace, deployment, service string) []string {
	var steps []string
	if deployment != "" {
		command := "kubectl describe deployment " + deployment
		if namespace != "" {
			command = "kubectl describe deployment -n " + namespace + " " + deployment
		}
		steps = append(steps, "Discovery follow-up: Try `"+command+"` if the event stream points to the owning deployment rather than only the pod.")
	}
	if service != "" {
		command := "kubectl describe service " + service
		if namespace != "" {
			command = "kubectl describe service -n " + namespace + " " + service
		}
		steps = append(steps, "Discovery follow-up: Try `"+command+"` if the event stream points to a service-level routing or selector issue.")
	}
	return steps
}

func isKubernetesContainerName(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func sanitizeDependencyHost(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.Trim(value, "`\"'()[]{}.,:;")
	if value == "" {
		return ""
	}
	switch value {
	case "to", "database", "server", "host", "hostname":
		return ""
	}
	if strings.Contains(value, "/") {
		return ""
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			continue
		}
		return ""
	}
	return value
}

func sanitizeDependencyPort(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return ""
	}
	return strconv.Itoa(port)
}

func interpretPS(command string, text string) models.Response {
	response := baseInterpretResponse(command)
	response.IntentID = "interpret_ps"
	response.ExpectedOutput = "A summary of whether any processes are consuming unusually high CPU or memory."

	var highCPU, highMem []string
	var parsed int

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "USER") || strings.HasPrefix(line, "PID") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		parsed++
		cpu, errCPU := strconv.ParseFloat(fields[2], 64)
		mem, errMem := strconv.ParseFloat(fields[3], 64)
		name := fields[10]
		if errCPU == nil && cpu >= 50 {
			highCPU = append(highCPU, fmt.Sprintf("%s (%.1f%% CPU)", name, cpu))
		}
		if errMem == nil && mem >= 20 {
			highMem = append(highMem, fmt.Sprintf("%s (%.1f%% MEM)", name, mem))
		}
	}

	if parsed == 0 {
		response.Explanation = "The pasted output did not look like standard `ps aux` output, so process-resource interpretation was limited."
		response.Confidence = "Low"
		response.Warnings = []string{"unable to parse process rows from provided output"}
		return response
	}

	if len(highCPU) == 0 && len(highMem) == 0 {
		response.Explanation = "No process in the pasted output is consuming more than 50% CPU or 20% memory."
		response.Confidence = "High"
		return response
	}

	var parts []string
	if len(highCPU) > 0 {
		parts = append(parts, "high CPU: "+strings.Join(highCPU, ", "))
	}
	if len(highMem) > 0 {
		parts = append(parts, "high memory: "+strings.Join(highMem, ", "))
	}
	response.Explanation = "At least one process is consuming significant resources: " + strings.Join(parts, "; ") + "."
	response.Confidence = "High"
	response.NextSteps = []string{
		"Use `lsof -p <PID>` to inspect open files for the heavy process.",
		"Check application logs for the offending process.",
	}
	return response
}

func interpretIPAddr(command string, text string) models.Response {
	response := baseInterpretResponse(command)
	response.IntentID = "interpret_ip_addr"
	response.ExpectedOutput = "A summary of interface states and whether any interfaces appear down or unconfigured."

	var downIfaces, noIP []string
	var currentIface string
	var hasIP bool
	var parsed int

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Interface header lines start with a digit (interface index).
		if len(trimmed) > 0 && trimmed[0] >= '1' && trimmed[0] <= '9' {
			// Save state of the previous interface.
			if currentIface != "" && !hasIP {
				noIP = append(noIP, currentIface)
			}
			hasIP = false
			parsed++
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				currentIface = strings.TrimSuffix(fields[1], ":")
			}
			lower := strings.ToLower(trimmed)
			if strings.Contains(lower, "state down") || strings.Contains(lower, ",down,") {
				downIfaces = append(downIfaces, currentIface)
			}
			continue
		}
		if strings.Contains(trimmed, "inet ") || strings.Contains(trimmed, "inet6 ") {
			hasIP = true
		}
	}
	// Final interface.
	if currentIface != "" && !hasIP {
		noIP = append(noIP, currentIface)
	}

	if parsed == 0 {
		response.Explanation = "The pasted output did not look like standard `ip addr` output."
		response.Confidence = "Low"
		response.Warnings = []string{"unable to parse interface blocks from provided output"}
		return response
	}

	var issues []string
	if len(downIfaces) > 0 {
		issues = append(issues, fmt.Sprintf("interface(s) DOWN: %s", strings.Join(downIfaces, ", ")))
	}
	if len(noIP) > 0 {
		issues = append(issues, fmt.Sprintf("no IP assigned: %s", strings.Join(noIP, ", ")))
	}

	if len(issues) == 0 {
		response.Explanation = "All parsed interfaces appear to be UP and have at least one IP address assigned."
		response.Confidence = "High"
		return response
	}

	response.Explanation = "The pasted ip addr output shows potential network issues: " + strings.Join(issues, "; ") + "."
	response.Confidence = "High"
	response.NextSteps = []string{
		"Run `ip link show` to inspect link-layer state for the affected interfaces.",
		"Check `/etc/NetworkManager/system-connections/` or `nmcli con show` for connection configuration.",
	}
	return response
}
