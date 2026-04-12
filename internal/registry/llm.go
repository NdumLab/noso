package registry

import (
	"fmt"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/troubleshoot"
	"github.com/NdumLab/noso/pkg/models"
)

func ResolveLLMCandidate(candidate models.LLMIntentCandidate, env models.Environment, collector evidence.Collector) (models.Response, bool, error) {
	var (
		response models.Response
		err      error
	)

	switch candidate.Intent {
	case "service_status":
		response, err = serviceIntent(fmt.Sprintf("%s service status", fallbackTarget(candidate.Target, "service")), collector)
	case "service_logs":
		response, err = serviceLogsIntent(fmt.Sprintf("logs for %s service", fallbackTarget(candidate.Target, "service")), collector)
	case "service_troubleshoot":
		response, _ = troubleshoot.Resolve(fmt.Sprintf("%s not starting", fallbackTarget(candidate.Target, "service")), collector)
	case "k8s_pod_status":
		response, err = kubectlDescribePodIntent(fmt.Sprintf("describe pod %s", fallbackTarget(candidate.Target, "pod-name")), collector)
	case "k8s_pod_logs":
		query := fmt.Sprintf("logs for pod %s", fallbackTarget(candidate.Target, "pod-name"))
		if candidate.Namespace != "" {
			query += " namespace " + candidate.Namespace
		}
		response, err = kubectlLogsIntent(query, collector)
	case "k8s_pod_troubleshoot":
		query := fmt.Sprintf("describe pod %s", fallbackTarget(candidate.Target, "pod-name"))
		if candidate.Namespace != "" {
			query += " namespace " + candidate.Namespace
		}
		response, err = kubectlDescribePodIntent(query, collector)
		if err == nil {
			response.IntentID = "troubleshoot_k8s_pod"
			response.Explanation = fmt.Sprintf("Start by describing pod %s to capture current status, events, and container state before changing the workload.", fallbackTarget(candidate.Target, "pod-name"))
			response.NextSteps = append(response.NextSteps,
				"Run `kubectl get events -A --sort-by=.metadata.creationTimestamp` to review recent scheduler and image-pull events.",
				"Run `kubectl logs <pod> --tail=100` if the pod is starting and emitting application logs.")
		}
	case "runtime_logs":
		response, err = runtimeLogsIntent(fmt.Sprintf("%s logs %s", fallbackTarget(candidate.ToolHint, "docker"), fallbackTarget(candidate.Target, "container-name")), collector)
	case "runtime_inspect":
		response, err = runtimeInspectIntent(fmt.Sprintf("inspect %s container %s", fallbackTarget(candidate.ToolHint, "docker"), fallbackTarget(candidate.Target, "container-name")), collector)
	case "runtime_troubleshoot":
		response, _ = troubleshoot.Resolve(fmt.Sprintf("%s container %s not starting", fallbackTarget(candidate.ToolHint, "docker"), fallbackTarget(candidate.Target, "container-name")), collector)
	case "git_push":
		response, err = gitPushIntent(collector)
	case "package_install":
		response, err = packageInstallIntent("install "+fallbackTarget(candidate.Target, "package-name"), env, collector)
	case "dns_lookup":
		response, err = dnsLookupIntent("nslookup "+fallbackTarget(candidate.Target, "example.com"), collector)
	case "cron_list":
		response, err = cronListIntent(collector)
	default:
		return models.Response{}, false, nil
	}

	if err != nil {
		return models.Response{}, false, err
	}
	return response, true, nil
}

func ClarificationResponse(question string, candidates []models.LLMIntentCandidate) models.Response {
	response := models.Response{
		IntentID:       "clarify_query",
		Explanation:    question,
		ExpectedOutput: "A narrower question that names the service, container runtime, or Kubernetes object to inspect.",
		Risk:           "Low",
		Confidence:     "Medium",
		NextSteps: []string{
			"Try re-running the query with the tool or object type, for example: `worker2 service status`, `logs for pod worker-2`, or `docker logs worker-2`.",
		},
	}
	for _, candidate := range candidates {
		response.Warnings = append(response.Warnings, fmt.Sprintf("candidate: %s target=%s confidence=%.2f", candidate.Intent, candidate.Target, candidate.Confidence))
	}
	return response
}

func fallbackTarget(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
