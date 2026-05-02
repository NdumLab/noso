package troubleshoot

import (
	"strings"

	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

func ResolveSuggestedTarget(state State, query string) (StateThread, SuggestedTarget, bool) {
	if len(state.Threads) == 0 {
		return StateThread{}, SuggestedTarget{}, false
	}
	normalized := normalizeQuery(query)
	for _, thread := range state.Threads {
		for _, suggestion := range thread.SuggestedTargets {
			if suggestion.Name != "" && strings.Contains(normalized, normalizeQuery(suggestion.Name)) {
				return thread, suggestion, true
			}
		}
	}
	return StateThread{}, SuggestedTarget{}, false
}

func ApplySuggestedTargetQuery(baseQuery string, suggestion SuggestedTarget) string {
	baseQuery = strings.TrimSpace(baseQuery)
	if baseQuery == "" {
		return suggestion.Name
	}
	lower := strings.ToLower(baseQuery)
	switch suggestion.Family {
	case "kubernetes":
		if strings.Contains(lower, "pod") || strings.Contains(lower, "kubernetes") || strings.Contains(lower, "kubectl") || strings.Contains(lower, "k8s") {
			return "pod " + suggestion.Name + " " + baseQuery
		}
		return "pod " + suggestion.Name + " " + baseQuery
	case "runtime":
		if strings.Contains(lower, "container") || strings.Contains(lower, "docker") || strings.Contains(lower, "podman") {
			return "container " + suggestion.Name + " " + baseQuery
		}
		return "container " + suggestion.Name + " " + baseQuery
	case "service":
		if strings.Contains(lower, "service") || strings.Contains(lower, "systemd") || strings.Contains(lower, "unit") {
			return "service " + suggestion.Name + " " + baseQuery
		}
		return "service " + suggestion.Name + " " + baseQuery
	case "kubernetes-node", "kubernetes-pvc", "kubernetes-secret", "kubernetes-configmap", "kubernetes-deployment", "kubernetes-service":
		return baseQuery
	default:
		return suggestion.Name + " " + baseQuery
	}
}

func ApplySuggestedTarget(thread StateThread, suggestion SuggestedTarget) StateThread {
	thread.LastCommand = ""
	thread.Executed = nil
	thread.LastDiscovery = nil
	thread.LastFindings = nil
	thread.SuggestedTargets = nil
	thread.History = nil
	thread.LastWarnings = []string{"operator adopted discovered target: " + suggestion.Name + " (" + suggestion.Family + ")"}
	if thread.FamilyScores == nil {
		thread.FamilyScores = map[string]float64{}
	}
	if thread.CauseScores == nil {
		thread.CauseScores = map[string]float64{}
	}
	thread.ActiveFamily = suggestion.Family
	thread.ActiveTarget = suggestion.Name
	thread.ActiveNamespace = suggestion.Namespace
	thread.RuntimeHint = ""
	switch suggestion.Family {
	case "kubernetes":
		thread.ActiveContainer = ""
		thread.RuntimeHint = ""
		thread.FamilyScores["kubernetes"] += 2.6
		thread.FamilyScores["service"] -= 1.2
		thread.FamilyScores["runtime"] -= 0.8
		clearCauseScores(thread.CauseScores, "service_unit_missing", "service_process_failure", "runtime_container_failure")
	case "runtime":
		thread.ActiveNamespace = ""
		thread.FamilyScores["runtime"] += 2.6
		thread.FamilyScores["service"] -= 1.1
		thread.FamilyScores["kubernetes"] -= 0.9
		thread.RuntimeHint = runtimeFromSuggestedCommand(suggestion.Command)
		thread.ActiveContainer = ""
		clearCauseScores(thread.CauseScores, "service_unit_missing", "service_process_failure", "kubernetes_crashloop", "kubernetes_image_pull", "kubernetes_scheduling_capacity")
	case "service":
		thread.FamilyScores["service"] += 2.6
		thread.FamilyScores["runtime"] -= 0.9
		thread.FamilyScores["kubernetes"] -= 0.9
		clearCauseScores(thread.CauseScores, "runtime_container_failure", "kubernetes_crashloop", "kubernetes_image_pull", "kubernetes_scheduling_capacity")
	case "kubernetes-node", "kubernetes-pvc", "kubernetes-secret", "kubernetes-configmap", "kubernetes-deployment", "kubernetes-service":
		thread.FamilyScores["kubernetes"] += 1.6
		thread.FamilyScores["service"] -= 0.8
		thread.FamilyScores["runtime"] -= 0.6
		thread.ActiveContainer = ""
		clearCauseScores(thread.CauseScores, "service_unit_missing", "service_process_failure", "runtime_container_failure")
	}
	return thread
}

func SuggestedTargetResponse(suggestion SuggestedTarget) (models.Response, bool) {
	command := strings.TrimSpace(suggestion.Command)
	if command == "" {
		return models.Response{}, false
	}
	base := models.Response{
		Command:    command,
		Risk:       safety.Classify(command),
		Confidence: "High",
	}
	switch suggestion.Family {
	case "kubernetes-node":
		base.IntentID = "inspect_k8s_node_describe"
		base.Explanation = "Inspect the discovered Kubernetes node directly because the troubleshoot thread found a node-specific scheduling signal."
		base.ExpectedOutput = "A detailed node report including labels, taints, allocatable resources, conditions, and recent events."
	case "kubernetes-pvc":
		base.IntentID = "inspect_k8s_pvc_describe"
		base.Explanation = "Inspect the discovered PersistentVolumeClaim directly because the troubleshoot thread found a storage-binding or mount blocker."
		base.ExpectedOutput = "A PVC description showing phase, storage class, bound PV, access modes, events, and recent binding failures."
	case "kubernetes-secret":
		base.IntentID = "inspect_k8s_secret_describe"
		base.Explanation = "Inspect the discovered Secret directly because the troubleshoot thread found a missing or mount-related secret reference."
		base.ExpectedOutput = "A Secret description showing metadata, type, and whether the object exists in the expected namespace."
	case "kubernetes-configmap":
		base.IntentID = "inspect_k8s_configmap_describe"
		base.Explanation = "Inspect the discovered ConfigMap directly because the troubleshoot thread found a missing or mount-related config reference."
		base.ExpectedOutput = "A ConfigMap description showing metadata and whether the object exists in the expected namespace."
	case "kubernetes-deployment":
		base.IntentID = "inspect_k8s_deployment_describe"
		base.Explanation = "Inspect the discovered Deployment directly because the troubleshoot thread found a rollout or owner-level workload signal."
		base.ExpectedOutput = "A Deployment description showing replicas, rollout conditions, selector state, and recent workload-level events."
	case "kubernetes-service":
		base.IntentID = "inspect_k8s_service_describe"
		base.Explanation = "Inspect the discovered Service directly because the troubleshoot thread found a service-level routing or owner signal."
		base.ExpectedOutput = "A Service description showing selectors, ports, endpoints, and whether traffic should currently resolve to healthy pods."
	default:
		return models.Response{}, false
	}
	return base, true
}

func runtimeFromSuggestedCommand(command string) string {
	switch {
	case strings.HasPrefix(command, "podman "):
		return "podman"
	case strings.HasPrefix(command, "docker "):
		return "docker"
	default:
		return ""
	}
}

func clearCauseScores(scores map[string]float64, ids ...string) {
	for _, id := range ids {
		delete(scores, id)
	}
}

func parseSuggestedTargets(steps []string) []SuggestedTarget {
	var suggestions []SuggestedTarget
	for _, step := range steps {
		if !strings.Contains(step, "Discovery follow-up: Try `") {
			continue
		}
		command := extractBacktickCommand(step)
		if command == "" {
			continue
		}
		fields := strings.Fields(command)
		if len(fields) == 0 {
			continue
		}
		suggestion := SuggestedTarget{Command: command}
		switch {
		case strings.HasPrefix(command, "systemctl status ") && len(fields) >= 3:
			suggestion.Family = "service"
			suggestion.Name = fields[2]
		case (strings.HasPrefix(command, "podman logs ") || strings.HasPrefix(command, "docker logs ")) && len(fields) >= 1:
			suggestion.Family = "runtime"
			suggestion.Name = fields[len(fields)-1]
		case strings.HasPrefix(command, "kubectl describe pod ") && len(fields) >= 4:
			suggestion.Family = "kubernetes"
			suggestion.Name, suggestion.Namespace = kubernetesDescribeTarget(fields, "pod")
		case strings.HasPrefix(command, "kubectl describe node ") && len(fields) >= 4:
			suggestion.Family = "kubernetes-node"
			suggestion.Name = fields[3]
		case strings.HasPrefix(command, "kubectl describe pvc ") && len(fields) >= 4:
			suggestion.Family = "kubernetes-pvc"
			suggestion.Name, suggestion.Namespace = kubernetesDescribeTarget(fields, "pvc")
		case strings.HasPrefix(command, "kubectl describe secret ") && len(fields) >= 4:
			suggestion.Family = "kubernetes-secret"
			suggestion.Name, suggestion.Namespace = kubernetesDescribeTarget(fields, "secret")
		case strings.HasPrefix(command, "kubectl describe configmap ") && len(fields) >= 4:
			suggestion.Family = "kubernetes-configmap"
			suggestion.Name, suggestion.Namespace = kubernetesDescribeTarget(fields, "configmap")
		case strings.HasPrefix(command, "kubectl describe deployment ") && len(fields) >= 4:
			suggestion.Family = "kubernetes-deployment"
			suggestion.Name, suggestion.Namespace = kubernetesDescribeTarget(fields, "deployment")
		case strings.HasPrefix(command, "kubectl describe service ") && len(fields) >= 4:
			suggestion.Family = "kubernetes-service"
			suggestion.Name, suggestion.Namespace = kubernetesDescribeTarget(fields, "service")
		}
		if suggestion.Family == "" || suggestion.Name == "" {
			continue
		}
		suggestions = append(suggestions, suggestion)
	}
	return suggestions
}

func kubernetesDescribeTarget(fields []string, kind string) (string, string) {
	namespace := ""
	name := ""
	for i := 0; i < len(fields); i++ {
		if fields[i] == "-n" && i+1 < len(fields) {
			namespace = fields[i+1]
			i++
			continue
		}
		if fields[i] == kind && i+1 < len(fields) {
			candidateIndex := i + 1
			if fields[candidateIndex] == "-n" && candidateIndex+2 < len(fields) {
				namespace = fields[candidateIndex+1]
				candidateIndex += 2
			}
			if candidateIndex < len(fields) {
				name = fields[candidateIndex]
			}
		}
	}
	if name == "" && len(fields) > 0 {
		name = fields[len(fields)-1]
	}
	return name, namespace
}
