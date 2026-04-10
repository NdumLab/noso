package interpret

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/noso-dev/noso/internal/safety"
	"github.com/noso-dev/noso/pkg/models"
)

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
	case strings.HasPrefix(lower, "df "):
		return interpretDF(command, text), nil
	case strings.HasPrefix(lower, "free "):
		return interpretFree(command, text), nil
	case strings.HasPrefix(lower, "kubectl get pods"):
		return interpretKubectlGetPods(command, text), nil
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
		if line == "" || strings.HasPrefix(line, "NAME ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		parsed++
		name := fields[0]
		status := fields[2]
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
	response := baseInterpretResponse(command)
	response.IntentID = "interpret_journalctl"
	response.ExpectedOutput = "A summary of error signals, failure patterns, and the next safe diagnostics."

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
	if strings.Contains(lower, "no space left") {
		signals = append(signals, "disk-full errors detected")
	}

	if len(signals) == 0 {
		if strings.TrimSpace(text) == "" {
			response.Explanation = "The pasted journalctl output appears empty. The unit may have no journal entries yet, or the time window returned no records."
			response.Confidence = "Low"
		} else {
			response.Explanation = "No obvious failure signals were found in the pasted journal output. The lines do not contain common error keywords."
			response.Confidence = "Medium"
			response.NextSteps = []string{
				"Scan the output manually for unusual state transitions or unexpected restarts.",
				"Widen the time window with `journalctl -u <service> --since today`.",
			}
		}
		return response
	}

	response.Explanation = fmt.Sprintf("The pasted journal output contains %d signal(s): %s.",
		len(signals), strings.Join(signals, "; "))
	response.Confidence = "High"
	response.NextSteps = []string{
		"Check the lines around each error for the root cause.",
		"Run `journalctl -u <service> -n 100 --no-pager` for more context.",
	}
	if strings.Contains(lower, "oom") || strings.Contains(lower, "killed process") {
		response.NextSteps = append(response.NextSteps,
			"Run `free -h` to inspect current memory state.",
			"Run `ps aux --sort=-%mem | head` to identify memory-heavy processes.")
	}
	return response
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
