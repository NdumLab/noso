package registry

import (
	"fmt"
	"strings"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

func helmVersionIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("helm")
	command := "helm version --short"
	response := models.Response{
		IntentID:       "inspect_helm_version",
		Command:        command,
		Explanation:    "Shows the installed Helm client version in a compact format.",
		ExpectedOutput: "A short Helm version string for the local client.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "helm")
	return response, nil
}

func helmReposIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("helm")
	command := "helm repo list"
	response := models.Response{
		IntentID:       "inspect_helm_repos",
		Command:        command,
		Explanation:    "Lists configured Helm chart repositories without modifying local state.",
		ExpectedOutput: "A table of configured Helm repository names and URLs.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "helm")
	return response, nil
}

func helmReleasesIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("helm")
	namespace := extractNamespace(query)
	command := "helm list -A"
	if namespace != "" {
		command = fmt.Sprintf("helm list -n %s", namespace)
	}
	response := models.Response{
		IntentID:       "inspect_helm_releases",
		Command:        command,
		Explanation:    "Lists Helm releases and their revision, status, chart, and app version details.",
		ExpectedOutput: "A release table showing installed release names, namespaces, revisions, and status.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "helm")
	return response, nil
}

func helmStatusIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("helm")
	release := extractNamedObject(query, "release", "release-name")
	namespace := extractNamespace(query)
	command := fmt.Sprintf("helm status %s", release)
	if namespace != "" {
		command = fmt.Sprintf("helm status %s -n %s", release, namespace)
	}
	response := models.Response{
		IntentID:       "inspect_helm_release_status",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows the current state, chart metadata, notes, and resource summary for Helm release %s.", release),
		ExpectedOutput: "Helm release status details including last deployment time, namespace, resources, and notes when available.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "helm")
	return response, nil
}

func helmHistoryIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("helm")
	release := extractNamedObject(query, "release", "release-name")
	namespace := extractNamespace(query)
	command := fmt.Sprintf("helm history %s", release)
	if namespace != "" {
		command = fmt.Sprintf("helm history %s -n %s", release, namespace)
	}
	response := models.Response{
		IntentID:       "inspect_helm_release_history",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows the revision history for Helm release %s without changing it.", release),
		ExpectedOutput: "A revision table showing revision numbers, update times, status, chart version, and descriptions.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "helm")
	return response, nil
}

func helmValuesIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("helm")
	release := extractNamedObject(query, "release", "release-name")
	namespace := extractNamespace(query)
	command := fmt.Sprintf("helm get values %s", release)
	if namespace != "" {
		command = fmt.Sprintf("helm get values %s -n %s", release, namespace)
	}
	response := models.Response{
		IntentID:       "inspect_helm_values",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows the user-supplied values for Helm release %s.", release),
		ExpectedOutput: "Rendered YAML values used for the selected release.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "helm")
	return response, nil
}

func helmTemplateIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("helm")
	chart := extractChartName(query)
	command := fmt.Sprintf("helm template %s", chart)
	response := models.Response{
		IntentID:       "preview_helm_template",
		Command:        command,
		Explanation:    fmt.Sprintf("Renders chart %s locally so you can preview Kubernetes manifests without installing or upgrading a release.", chart),
		ExpectedOutput: "YAML manifests printed to stdout, representing the rendered templates for the chart.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "helm")
	return response, nil
}

func extractChartName(query string) string {
	lower := strings.ToLower(query)
	if idx := strings.Index(lower, "chart "); idx >= 0 {
		fields := strings.Fields(query[idx+len("chart "):])
		if len(fields) > 0 {
			return strings.Trim(fields[0], "`\"'")
		}
	}
	if idx := strings.Index(lower, "template "); idx >= 0 {
		fields := strings.Fields(query[idx+len("template "):])
		if len(fields) > 0 {
			return strings.Trim(fields[0], "`\"'")
		}
	}
	return "chart-dir"
}
