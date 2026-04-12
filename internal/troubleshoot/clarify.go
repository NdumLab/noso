package troubleshoot

import "strings"

type ClarificationHint struct {
	Family string
	Label  string
}

func ResolveClarification(state State, query string) (StateThread, ClarificationHint, bool) {
	hint, ok := parseClarificationHint(query)
	if !ok || len(state.Threads) == 0 {
		return StateThread{}, ClarificationHint{}, false
	}
	return state.Threads[0], hint, true
}

func ApplyClarificationQuery(baseQuery string, hint ClarificationHint) string {
	baseQuery = strings.TrimSpace(baseQuery)
	if baseQuery == "" {
		return hint.Label
	}
	lower := strings.ToLower(baseQuery)
	switch hint.Family {
	case "kubernetes":
		if strings.Contains(lower, "pod") || strings.Contains(lower, "kubernetes") || strings.Contains(lower, "kubectl") || strings.Contains(lower, "k8s") {
			return baseQuery
		}
	case "runtime":
		if strings.Contains(lower, "container") || strings.Contains(lower, "docker") || strings.Contains(lower, "podman") || strings.Contains(lower, "containerd") {
			return baseQuery
		}
	case "service":
		if strings.Contains(lower, "service") || strings.Contains(lower, "systemd") || strings.Contains(lower, "unit") {
			return baseQuery
		}
	}
	return strings.TrimSpace(baseQuery + " " + hint.Label)
}

func ApplyClarificationHint(thread StateThread, hint ClarificationHint) StateThread {
	if thread.FamilyScores == nil {
		thread.FamilyScores = map[string]float64{}
	}
	if thread.CauseScores == nil {
		thread.CauseScores = map[string]float64{}
	}
	thread.ActiveFamily = hint.Family
	if hint.Family != "kubernetes" {
		thread.ActiveNamespace = ""
	}
	if hint.Family != "runtime" {
		thread.RuntimeHint = ""
	}
	switch hint.Family {
	case "kubernetes":
		thread.FamilyScores["kubernetes"] += 2.5
		thread.FamilyScores["runtime"] -= 0.8
		thread.FamilyScores["service"] -= 1.4
		adjustCauseScore(thread.CauseScores, "service_unit_missing", -2.5)
		adjustCauseScore(thread.CauseScores, "service_process_failure", -2.0)
		adjustCauseScore(thread.CauseScores, "runtime_container_failure", -0.6)
	case "runtime":
		thread.FamilyScores["runtime"] += 2.5
		thread.FamilyScores["kubernetes"] -= 0.9
		thread.FamilyScores["service"] -= 1.2
		adjustCauseScore(thread.CauseScores, "service_unit_missing", -2.5)
		adjustCauseScore(thread.CauseScores, "service_process_failure", -1.6)
		adjustCauseScore(thread.CauseScores, "kubernetes_crashloop", -1.2)
		adjustCauseScore(thread.CauseScores, "kubernetes_image_pull", -1.0)
		adjustCauseScore(thread.CauseScores, "kubernetes_scheduling_capacity", -1.0)
	case "service":
		thread.FamilyScores["service"] += 2.5
		thread.FamilyScores["runtime"] -= 0.8
		thread.FamilyScores["kubernetes"] -= 0.8
		adjustCauseScore(thread.CauseScores, "runtime_container_failure", -1.5)
		adjustCauseScore(thread.CauseScores, "kubernetes_crashloop", -1.5)
		adjustCauseScore(thread.CauseScores, "kubernetes_image_pull", -1.2)
		adjustCauseScore(thread.CauseScores, "kubernetes_scheduling_capacity", -1.2)
	}
	if thread.LastWarnings == nil {
		thread.LastWarnings = nil
	}
	thread.LastWarnings = appendUniqueStrings(thread.LastWarnings, "operator clarification: treat this thread as a "+hint.Label+" problem")
	return thread
}

func parseClarificationHint(query string) (ClarificationHint, bool) {
	normalized := normalizeQuery(query)
	if normalized == "" {
		return ClarificationHint{}, false
	}
	if !looksLikeClarification(normalized) {
		return ClarificationHint{}, false
	}
	switch {
	case containsAny(normalized, "actually a pod", "actually pod", "it is a pod", "its a pod", "it is kubernetes", "its kubernetes", "it is k8s", "its k8s", "this is a pod", "this is kubernetes"):
		return ClarificationHint{Family: "kubernetes", Label: "pod"}, true
	case containsAny(normalized, "actually a container", "actually container", "it is a container", "its a container", "this is a container", "actually docker", "actually podman", "docker container", "podman container"):
		return ClarificationHint{Family: "runtime", Label: "container"}, true
	case containsAny(normalized, "actually a service", "actually service", "it is a service", "its a service", "this is a service", "it is systemd", "its systemd", "it is a unit", "its a unit"):
		return ClarificationHint{Family: "service", Label: "service"}, true
	default:
		return ClarificationHint{}, false
	}
}

func looksLikeClarification(normalized string) bool {
	return containsAny(normalized,
		"actually",
		"it is",
		"its",
		"this is",
		"i mean",
		"meant",
	)
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
