package registry

import (
	"fmt"
	"strings"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/internal/safety"
	"github.com/noso-dev/noso/pkg/models"
)

func runtimeStatusIntent(query string, collector evidence.Collector) (models.Response, error) {
	runtime := detectRuntime(query, "docker")
	ev := collector.Lookup("systemctl")
	command := fmt.Sprintf("systemctl status %s --no-pager -l", runtime)
	response := models.Response{
		IntentID:       "inspect_runtime_status",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows whether the %s service is loaded and active, plus recent unit details and failure context.", runtime),
		ExpectedOutput: fmt.Sprintf("Loaded or active state, the main PID, recent log lines, and any recent failure information for the %s service.", runtime),
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "systemctl")
	addRuntimeMissingWarning(&response, runtime, collector)
	return response, nil
}

func runtimeVersionIntent(query string, collector evidence.Collector) (models.Response, error) {
	runtime := detectRuntime(query, "docker")
	ev := collector.Lookup(runtime)
	command := fmt.Sprintf("%s version", runtime)
	response := models.Response{
		IntentID:       "inspect_runtime_version",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows client and server version details for %s when the runtime is installed.", runtime),
		ExpectedOutput: "Version output for the runtime client and, when available, the connected server or daemon.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, runtime)
	return response, nil
}

func runtimePsIntent(query string, collector evidence.Collector) (models.Response, error) {
	runtime := detectRuntime(query, "docker")
	ev := collector.Lookup(runtime)
	command := fmt.Sprintf("%s ps -a", runtime)
	response := models.Response{
		IntentID:       "inspect_runtime_containers",
		Command:        command,
		Explanation:    fmt.Sprintf("Lists running and stopped containers known to %s without changing runtime state.", runtime),
		ExpectedOutput: "A container table with IDs, images, names, statuses, and exposed ports when available.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, runtime)
	return response, nil
}

func runtimeImagesIntent(query string, collector evidence.Collector) (models.Response, error) {
	runtime := detectRuntime(query, "docker")
	ev := collector.Lookup(runtime)
	command := fmt.Sprintf("%s images", runtime)
	response := models.Response{
		IntentID:       "inspect_runtime_images",
		Command:        command,
		Explanation:    fmt.Sprintf("Lists images available to %s, including repositories, tags, IDs, and sizes.", runtime),
		ExpectedOutput: "An image table with repository, tag, image ID, creation time, and size.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, runtime)
	return response, nil
}

func runtimeLogsIntent(query string, collector evidence.Collector) (models.Response, error) {
	runtime := detectRuntime(query, "docker")
	target := "container-name"
	if matches := runtimeLogsRegex.FindStringSubmatch(strings.ToLower(query)); len(matches) >= 3 {
		if matches[1] != "" {
			target = matches[1]
		}
		if matches[2] != "" {
			target = matches[2]
		}
	}
	ev := collector.Lookup(runtime)
	command := fmt.Sprintf("%s logs --tail 100 %s", runtime, target)
	response := models.Response{
		IntentID:       "inspect_runtime_container_logs",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows the last 100 log lines for container %s using %s.", target, runtime),
		ExpectedOutput: "Recent stdout or stderr log lines from the requested container, if it exists.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, runtime)
	return response, nil
}

func runtimeInspectIntent(query string, collector evidence.Collector) (models.Response, error) {
	runtime := "docker"
	target := "container-name"
	if matches := runtimeInspectRegex.FindStringSubmatch(strings.ToLower(query)); len(matches) == 3 {
		runtime = matches[1]
		target = matches[2]
	}
	ev := collector.Lookup(runtime)
	command := fmt.Sprintf("%s inspect %s", runtime, target)
	response := models.Response{
		IntentID:       "inspect_runtime_container",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows low-level JSON metadata for container %s, including mounts, network settings, and restart information.", target),
		ExpectedOutput: "A JSON document describing the container configuration and current runtime state.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, runtime)
	return response, nil
}

func detectRuntime(query, fallback string) string {
	normalized := strings.ToLower(query)
	switch {
	case strings.Contains(normalized, "podman"):
		return "podman"
	case strings.Contains(normalized, "docker"):
		return "docker"
	default:
		return fallback
	}
}

func addRuntimeMissingWarning(response *models.Response, runtime string, collector evidence.Collector) {
	if ev := collector.Lookup(runtime); !ev.Exists {
		appendWarning(response, runtime+" is not currently installed on this host")
	}
}
