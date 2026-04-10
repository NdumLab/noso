package registry

import (
	"fmt"
	"strings"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/internal/safety"
	"github.com/noso-dev/noso/pkg/models"
)

func kubectlVersionIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("kubectl")
	command := "kubectl version --client"
	response := models.Response{
		IntentID:       "inspect_k8s_version",
		Command:        command,
		Explanation:    "Shows the installed kubectl client version without requiring a cluster mutation.",
		ExpectedOutput: "Client version details for the local kubectl binary.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "kubectl")
	return response, nil
}

func kubectlContextIntent(env models.Environment, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("kubectl")
	command := "kubectl config current-context"
	explanation := "Shows the active Kubernetes context so you can confirm which cluster and credentials kubectl will target."
	expected := "The current kubeconfig context name."
	response := models.Response{
		IntentID:       "inspect_k8s_context",
		Command:        command,
		Explanation:    explanation,
		ExpectedOutput: expected,
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "kubectl")
	if env.KubeContext == "" {
		appendWarning(&response, "no current Kubernetes context is available on this host")
	}
	return response, nil
}

func kubectlPodsIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("kubectl")
	namespace := extractNamespace(query)
	command := "kubectl get pods -A"
	if namespace != "" {
		command = fmt.Sprintf("kubectl get pods -n %s", namespace)
	}
	response := models.Response{
		IntentID:       "inspect_k8s_pods",
		Command:        command,
		Explanation:    "Lists pods and their readiness, status, restart counts, and age.",
		ExpectedOutput: "A pod table showing namespace, name, readiness, phase, restarts, and age.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "kubectl")
	return response, nil
}

func kubectlDeploymentsIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("kubectl")
	namespace := extractNamespace(query)
	command := "kubectl get deployments -A"
	if namespace != "" {
		command = fmt.Sprintf("kubectl get deployments -n %s", namespace)
	}
	response := models.Response{
		IntentID:       "inspect_k8s_deployments",
		Command:        command,
		Explanation:    "Lists deployments and their desired, current, and available replica counts.",
		ExpectedOutput: "A deployment table showing readiness, up-to-date replicas, and availability.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "kubectl")
	return response, nil
}

func kubectlServicesIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("kubectl")
	namespace := extractNamespace(query)
	command := "kubectl get services -A"
	if namespace != "" {
		command = fmt.Sprintf("kubectl get services -n %s", namespace)
	}
	response := models.Response{
		IntentID:       "inspect_k8s_services",
		Command:        command,
		Explanation:    "Lists services, types, cluster IPs, external IPs, and exposed ports.",
		ExpectedOutput: "A service table showing the service type and relevant IP or port information.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "kubectl")
	return response, nil
}

func kubectlNamespacesIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("kubectl")
	command := "kubectl get namespaces"
	response := models.Response{
		IntentID:       "inspect_k8s_namespaces",
		Command:        command,
		Explanation:    "Lists namespaces and their lifecycle state.",
		ExpectedOutput: "A namespace table showing each namespace and whether it is active or terminating.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "kubectl")
	return response, nil
}

func kubectlLogsIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("kubectl")
	pod := extractNamedObject(query, "pod", "pod-name")
	namespace := extractNamespace(query)
	command := fmt.Sprintf("kubectl logs %s --tail=100", pod)
	if namespace != "" {
		command = fmt.Sprintf("kubectl logs -n %s %s --tail=100", namespace, pod)
	}
	response := models.Response{
		IntentID:       "inspect_k8s_logs",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows the most recent 100 log lines for pod %s.", pod),
		ExpectedOutput: "Recent application log lines from the selected pod or container.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "kubectl")
	return response, nil
}

func kubectlDescribePodIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("kubectl")
	pod := extractNamedObject(query, "pod", "pod-name")
	namespace := extractNamespace(query)
	command := fmt.Sprintf("kubectl describe pod %s", pod)
	if namespace != "" {
		command = fmt.Sprintf("kubectl describe pod -n %s %s", namespace, pod)
	}
	response := models.Response{
		IntentID:       "inspect_k8s_pod_describe",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows detailed pod status, events, container states, mounts, and scheduling information for pod %s.", pod),
		ExpectedOutput: "A detailed pod report including conditions, events, container states, and related node information.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "kubectl")
	return response, nil
}

func kubectlEventsIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("kubectl")
	namespace := extractNamespace(query)
	command := "kubectl get events -A --sort-by=.metadata.creationTimestamp"
	if namespace != "" {
		command = fmt.Sprintf("kubectl get events -n %s --sort-by=.metadata.creationTimestamp", namespace)
	}
	response := models.Response{
		IntentID:       "inspect_k8s_events",
		Command:        command,
		Explanation:    "Shows recent cluster events in chronological order, which helps explain scheduling and image-pull failures.",
		ExpectedOutput: "An events table with involved objects, reasons, timestamps, and controller messages.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "kubectl")
	return response, nil
}

func extractNamespace(query string) string {
	lower := strings.ToLower(query)
	if idx := strings.Index(lower, "namespace "); idx >= 0 {
		fields := strings.Fields(lower[idx+len("namespace "):])
		if len(fields) > 0 {
			return strings.Trim(fields[0], "`\"'")
		}
	}
	return ""
}

func extractNamedObject(query, keyword, fallback string) string {
	lower := strings.ToLower(query)
	if idx := strings.Index(lower, keyword+" "); idx >= 0 {
		fields := strings.Fields(lower[idx+len(keyword+" "):])
		if len(fields) > 0 {
			return strings.Trim(fields[0], "`\"'")
		}
	}
	return fallback
}
