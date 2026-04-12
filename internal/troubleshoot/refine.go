package troubleshoot

import "strings"

type QueryRefinement struct {
	Namespace string
	Runtime   string
}

func ResolveThreadRefinement(state State, query string) (StateThread, QueryRefinement, bool) {
	if len(state.Threads) == 0 {
		return StateThread{}, QueryRefinement{}, false
	}
	thread := state.Threads[0]
	refinement := parseRefinement(query, thread)
	if refinement.Namespace == "" && refinement.Runtime == "" {
		return StateThread{}, QueryRefinement{}, false
	}
	return thread, refinement, true
}

func ApplyThreadRefinementQuery(baseQuery string, thread StateThread, refinement QueryRefinement) string {
	if thread.ActiveTarget == "" || thread.ActiveFamily == "" {
		return appendRefinement(baseQuery, refinement)
	}
	switch thread.ActiveFamily {
	case "kubernetes":
		query := "pod " + thread.ActiveTarget + " " + thread.Query
		if refinement.Namespace != "" {
			query += " namespace " + refinement.Namespace
		} else if thread.ActiveNamespace != "" {
			query += " namespace " + thread.ActiveNamespace
		}
		return strings.TrimSpace(query)
	case "kubernetes-pvc":
		return strings.TrimSpace(kubernetesObjectQuery("pvc", thread, refinement))
	case "kubernetes-secret":
		return strings.TrimSpace(kubernetesObjectQuery("secret", thread, refinement))
	case "kubernetes-configmap":
		return strings.TrimSpace(kubernetesObjectQuery("configmap", thread, refinement))
	case "kubernetes-deployment":
		return strings.TrimSpace(kubernetesObjectQuery("deployment", thread, refinement))
	case "kubernetes-service":
		return strings.TrimSpace(kubernetesObjectQuery("service", thread, refinement))
	case "kubernetes-node":
		return strings.TrimSpace("node " + thread.ActiveTarget + " " + thread.Query)
	case "runtime":
		runtime := refinement.Runtime
		if runtime == "" {
			runtime = thread.RuntimeHint
		}
		if runtime == "" {
			runtime = "docker"
		}
		return strings.TrimSpace("container " + thread.ActiveTarget + " " + thread.Query + " " + runtime)
	case "service":
		return strings.TrimSpace("service " + thread.ActiveTarget + " " + thread.Query)
	default:
		return appendRefinement(baseQuery, refinement)
	}
}

func ApplyThreadRefinement(thread StateThread, refinement QueryRefinement) StateThread {
	if refinement.Namespace != "" {
		thread.ActiveNamespace = refinement.Namespace
		thread.LastWarnings = appendUniqueStrings(thread.LastWarnings, "operator refinement: namespace "+refinement.Namespace)
	}
	if refinement.Runtime != "" {
		thread.RuntimeHint = refinement.Runtime
		thread.ActiveFamily = "runtime"
		thread.LastWarnings = appendUniqueStrings(thread.LastWarnings, "operator refinement: runtime "+refinement.Runtime)
	}
	return thread
}

func parseRefinement(query string, thread StateThread) QueryRefinement {
	normalized := normalizeQuery(query)
	if normalized == "" {
		return QueryRefinement{}
	}
	refinement := QueryRefinement{
		Namespace: extractRefinementNamespace(normalized, thread),
		Runtime:   extractRefinementRuntime(normalized),
	}
	return refinement
}

func extractRefinementRuntime(normalized string) string {
	switch {
	case strings.Contains(normalized, "podman"):
		return "podman"
	case strings.Contains(normalized, "docker"):
		return "docker"
	default:
		return ""
	}
}

func extractRefinementNamespace(normalized string, thread StateThread) string {
	if idx := strings.Index(normalized, "namespace "); idx >= 0 {
		fields := strings.Fields(normalized[idx+len("namespace "):])
		if len(fields) > 0 && isNameLike(fields[0]) {
			return fields[0]
		}
	}
	switch thread.ActiveFamily {
	case "kubernetes", "kubernetes-pvc", "kubernetes-secret", "kubernetes-configmap", "kubernetes-deployment", "kubernetes-service":
	default:
		return ""
	}
	fields := strings.Fields(normalized)
	if len(fields) <= 1 || len(fields) > 4 {
		return ""
	}
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == "in" && isNameLike(fields[i+1]) {
			return fields[i+1]
		}
	}
	return ""
}

func kubernetesObjectQuery(kind string, thread StateThread, refinement QueryRefinement) string {
	query := kind + " " + thread.ActiveTarget + " " + thread.Query
	if refinement.Namespace != "" {
		query += " namespace " + refinement.Namespace
	} else if thread.ActiveNamespace != "" {
		query += " namespace " + thread.ActiveNamespace
	}
	return query
}

func appendRefinement(baseQuery string, refinement QueryRefinement) string {
	baseQuery = strings.TrimSpace(baseQuery)
	if refinement.Runtime != "" {
		baseQuery = strings.TrimSpace(baseQuery + " " + refinement.Runtime)
	}
	if refinement.Namespace != "" {
		baseQuery = strings.TrimSpace(baseQuery + " namespace " + refinement.Namespace)
	}
	return baseQuery
}

func isNameLike(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}
