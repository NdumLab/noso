package registry

import (
	"strings"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

func containerdStatusIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("systemctl")
	command := "systemctl status containerd --no-pager -l"
	response := models.Response{
		IntentID:       "inspect_containerd_status",
		Command:        command,
		Explanation:    "Shows whether the containerd service is loaded and active, plus recent unit details and failure context.",
		ExpectedOutput: "Loaded/active state, the main PID, recent log lines, and any recent failure information for the containerd service.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "systemctl")
	if containerdEv := collector.Lookup("containerd"); !containerdEv.Exists {
		appendWarning(&response, "containerd is not currently installed on this host")
	}
	return response, nil
}

func containerdLogsIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("journalctl")
	command := "journalctl -u containerd -n 50 --no-pager"
	response := models.Response{
		IntentID:       "inspect_containerd_logs",
		Command:        command,
		Explanation:    "Shows the last 50 journal lines for the containerd service without changing runtime state.",
		ExpectedOutput: "Recent containerd log lines, including startup messages, warnings, and errors if the service exists.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "journalctl")
	if containerdEv := collector.Lookup("containerd"); !containerdEv.Exists {
		appendWarning(&response, "containerd is not currently installed on this host")
	}
	return response, nil
}

func containerdVersionIntent(collector evidence.Collector) (models.Response, error) {
	primary := collector.Lookup("containerd")
	command := "containerd --version"
	verified := append([]string{}, primary.VerificationSources...)
	confidence := confidenceFor(primary)
	warnings := []string{}

	if !primary.Exists {
		if ctrEv := collector.Lookup("ctr"); ctrEv.Exists {
			command = "ctr version"
			verified = append([]string{}, ctrEv.VerificationSources...)
			confidence = confidenceFor(ctrEv)
		} else if crictlEv := collector.Lookup("crictl"); crictlEv.Exists {
			command = "crictl version"
			verified = append([]string{}, crictlEv.VerificationSources...)
			confidence = confidenceFor(crictlEv)
		} else if nerdctlEv := collector.Lookup("nerdctl"); nerdctlEv.Exists {
			command = "nerdctl version"
			verified = append([]string{}, nerdctlEv.VerificationSources...)
			confidence = confidenceFor(nerdctlEv)
		} else {
			warnings = append(warnings, "containerd is not currently installed on this host")
		}
	}

	response := models.Response{
		IntentID:       "inspect_containerd_version",
		Command:        command,
		Explanation:    "Shows the available containerd runtime or client version using the first installed containerd-related tool found on the host.",
		ExpectedOutput: "Version output for containerd itself or an available client such as ctr, crictl, or nerdctl.",
		Risk:           safety.Classify(command),
		Confidence:     confidence,
		VerifiedFrom:   verified,
		Warnings:       warnings,
	}

	switch {
	case strings.HasPrefix(command, "containerd"):
		addHelpEvidence(&response, primary, "containerd")
	case strings.HasPrefix(command, "ctr"):
		addHelpEvidence(&response, collector.Lookup("ctr"), "ctr")
	case strings.HasPrefix(command, "crictl"):
		addHelpEvidence(&response, collector.Lookup("crictl"), "crictl")
	case strings.HasPrefix(command, "nerdctl"):
		addHelpEvidence(&response, collector.Lookup("nerdctl"), "nerdctl")
	}

	return response, nil
}
