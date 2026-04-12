package registry

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/troubleshoot"
	"github.com/NdumLab/noso/pkg/models"
)

type troubleshootHypothesis struct {
	Candidate models.LLMIntentCandidate
	Score     float64
	Reason    string
}

type targetDiscovery struct {
	ServiceFound         bool
	RuntimeFound         bool
	PodFound             bool
	RuntimeTool          string
	RuntimeHintRequested bool
	RuntimeToolAvailable bool
	RequestedNamespace   string
	PodFoundInNamespace  bool
	PodNamespaces        []string
	ServiceMatches       []string
	RuntimeMatches       []string
	PodMatches           []string
}

var discoverTargetKinds = liveDiscoverTargetKindsForQuery

// TroubleshootPlan builds a ranked, read-only troubleshoot plan for ambiguous
// operational questions. It reuses the deterministic registry and template
// responders instead of allowing the model to invent commands.
func TroubleshootPlan(query string, env models.Environment, collector evidence.Collector) (models.Response, bool, error) {
	if response, ok := troubleshoot.Resolve(query, collector); ok {
		return response, true, nil
	}

	hypotheses := heuristicTroubleshootHypotheses(query, env, collector)
	if len(hypotheses) == 0 {
		return models.Response{}, false, nil
	}
	return buildTroubleshootPlan(query, env, collector, hypotheses)
}

func TroubleshootPlanFromCandidates(query string, env models.Environment, collector evidence.Collector, candidates []models.LLMIntentCandidate) (models.Response, bool, error) {
	hypotheses := hypothesesFromCandidates(candidates)
	if len(hypotheses) == 0 {
		return models.Response{}, false, nil
	}
	return buildTroubleshootPlan(query, env, collector, hypotheses)
}

func buildTroubleshootPlan(query string, env models.Environment, collector evidence.Collector, hypotheses []troubleshootHypothesis) (models.Response, bool, error) {
	sort.SliceStable(hypotheses, func(i, j int) bool {
		return hypotheses[i].Score > hypotheses[j].Score
	})

	var resolved []models.Response
	var kept []troubleshootHypothesis
	for _, hypothesis := range hypotheses {
		response, ok, err := ResolveLLMCandidate(hypothesis.Candidate, env, collector)
		if err != nil {
			return models.Response{}, false, err
		}
		if !ok {
			continue
		}
		resolved = append(resolved, response)
		kept = append(kept, hypothesis)
	}
	if len(resolved) == 0 {
		return models.Response{}, false, nil
	}

	top := resolved[0]
	topHypothesis := kept[0]
	discovery := discoverTargetKinds(query, env, collector)
	target := inferTroubleshootTarget(query)
	response := models.Response{
		IntentID:       "troubleshoot_plan",
		Command:        top.Command,
		Explanation:    troubleshootExplanation(query, topHypothesis, len(kept)),
		ExpectedOutput: "The first probe should confirm the object type and surface the immediate failure reason before any restart, rollout, or package change.",
		Risk:           top.Risk,
		Confidence:     planConfidence(topHypothesis.Score, len(kept)),
		Discovery:      formatDiscoveryEvidence(discovery, target, env, collector),
		VerifiedFrom:   append([]string{}, top.VerifiedFrom...),
	}

	response.VerifiedFrom = append(response.VerifiedFrom, planVerificationSources(query, kept, env)...)
	response.NextSteps = append(response.NextSteps, formatHypothesisSteps(kept, resolved)...)
	response.NextSteps = append(response.NextSteps, discoveryFollowUpSteps(discovery)...)
	if len(top.NextSteps) > 0 {
		response.NextSteps = append(response.NextSteps, "After the primary probe:")
		for _, step := range top.NextSteps {
			response.NextSteps = append(response.NextSteps, "  "+step)
		}
	}
	if len(kept) > 1 {
		response.Warnings = append(response.Warnings, "query was ambiguous, so noso built a ranked troubleshoot plan instead of guessing a single object type")
	}
	return response, true, nil
}

func heuristicTroubleshootHypotheses(query string, env models.Environment, collector evidence.Collector) []troubleshootHypothesis {
	normalized := strings.ToLower(strings.TrimSpace(query))
	target := inferTroubleshootTarget(query)

	if !looksLikeTroubleshootQuery(normalized) {
		return nil
	}

	var hypotheses []troubleshootHypothesis
	appendUnique := func(candidate models.LLMIntentCandidate, score float64, reason string) {
		for _, existing := range hypotheses {
			if existing.Candidate.Intent == candidate.Intent && existing.Candidate.Target == candidate.Target && existing.Candidate.ToolHint == candidate.ToolHint && existing.Candidate.Namespace == candidate.Namespace {
				return
			}
		}
		hypotheses = append(hypotheses, troubleshootHypothesis{
			Candidate: candidate,
			Score:     score,
			Reason:    reason,
		})
	}

	switch {
	case strings.Contains(normalized, "disk full"), strings.Contains(normalized, "no space left"):
		appendUnique(models.LLMIntentCandidate{Intent: "service_troubleshoot", Target: target}, 0.30, "fallback service hypothesis kept only because the query uses generic outage language")
	case strings.Contains(normalized, "connection refused"), strings.Contains(normalized, "cannot connect"), strings.Contains(normalized, "can't connect"):
		appendUnique(models.LLMIntentCandidate{Intent: "service_troubleshoot", Target: target}, 0.35, "service startup remains the most common reason a local listener is absent")
	}

	if mentionsService(normalized) || (toolAvailable(env, collector, "systemctl") && target != "" && !mentionsContainerRuntime(normalized) && !mentionsKubernetes(normalized)) {
		score := 0.78
		reason := "systemd is available and the query looks like a service or unit outage"
		if strings.Contains(normalized, "not up") || strings.Contains(normalized, "down") {
			score = 0.82
			reason = "a worker-like name plus outage wording most often maps to a systemd service on this host"
		}
		appendUnique(models.LLMIntentCandidate{Intent: "service_troubleshoot", Target: target}, score, reason)
		appendUnique(models.LLMIntentCandidate{Intent: "service_status", Target: target}, score-0.06, "a direct status probe is the safest first read-only command")
	}

	if mentionsContainerRuntime(normalized) || (runtimeTool(env, collector) != "" && target != "" && !mentionsKubernetes(normalized)) {
		runtime := runtimeTool(env, collector)
		if runtime == "" {
			runtime = "docker"
		}
		score := 0.62
		reason := runtime + " is available and the query could refer to a container workload"
		if mentionsContainerRuntime(normalized) {
			score = 0.84
			reason = "the query explicitly mentions a container runtime"
		}
		appendUnique(models.LLMIntentCandidate{Intent: "runtime_troubleshoot", Target: target, ToolHint: runtime}, score, reason)
		appendUnique(models.LLMIntentCandidate{Intent: "runtime_logs", Target: target, ToolHint: runtime}, score-0.08, "container logs are the fastest follow-up once the runtime hypothesis is confirmed")
	}

	if mentionsKubernetes(normalized) || (toolAvailable(env, collector, "kubectl") && target != "") {
		k8sTarget := target
		namespace := extractNamespace(query)
		if strings.Contains(target, " ") {
			k8sTarget = strings.ReplaceAll(target, " ", "-")
		}
		score := 0.55
		reason := "kubectl is available and the target could be a pod or workload name"
		if mentionsKubernetes(normalized) {
			score = 0.86
			reason = "the query explicitly mentions Kubernetes or pod semantics"
		}
		appendUnique(models.LLMIntentCandidate{Intent: "k8s_pod_troubleshoot", Target: k8sTarget, Namespace: namespace}, score, reason)
		appendUnique(models.LLMIntentCandidate{Intent: "k8s_pod_logs", Target: k8sTarget, Namespace: namespace}, score-0.09, "pod logs are usually the next confirming signal after describe output")
	}

	applyTargetDiscovery(hypotheses, discoverTargetKinds(query, env, collector))

	return hypotheses
}

func formatDiscoveryEvidence(discovery targetDiscovery, target string, env models.Environment, collector evidence.Collector) []string {
	if !safeDiscoveryTarget(target) {
		return nil
	}
	var items []string
	primary := strings.ToLower(strings.TrimSpace(target))
	if discovery.ServiceFound {
		items = append(items, "Found matching systemd unit name for "+primary+".")
	} else if toolAvailable(env, collector, "systemctl") {
		items = append(items, "No matching systemd unit name found for "+primary+".")
		if len(discovery.ServiceMatches) > 0 {
			items = append(items, "Closest systemd unit names: "+strings.Join(discovery.ServiceMatches, ", ")+".")
		}
	}
	if discovery.RuntimeHintRequested && discovery.RuntimeTool != "" {
		if discovery.RuntimeToolAvailable {
			items = append(items, "Runtime hint confirmed: "+discovery.RuntimeTool+" is available on this host.")
		} else {
			items = append(items, "Runtime hint not confirmed: "+discovery.RuntimeTool+" is not available on this host.")
		}
	}
	if runtime := discovery.RuntimeTool; runtime == "podman" || runtime == "docker" {
		if discovery.RuntimeFound {
			items = append(items, "Found matching "+runtime+" container name for "+primary+".")
		} else {
			items = append(items, "No matching "+runtime+" container name found for "+primary+".")
			if len(discovery.RuntimeMatches) > 0 {
				items = append(items, "Closest "+runtime+" container names: "+strings.Join(discovery.RuntimeMatches, ", ")+".")
			}
		}
	}
	if toolAvailable(env, collector, "kubectl") {
		if discovery.PodFound {
			if discovery.RequestedNamespace != "" && discovery.PodFoundInNamespace {
				items = append(items, "Found matching Kubernetes pod name for "+primary+" in namespace "+discovery.RequestedNamespace+".")
			} else {
				items = append(items, "Found matching Kubernetes pod name for "+primary+".")
			}
		} else {
			if discovery.RequestedNamespace != "" {
				items = append(items, "No matching Kubernetes pod name found for "+primary+" in namespace "+discovery.RequestedNamespace+".")
			} else {
				items = append(items, "No matching Kubernetes pod name found for "+primary+".")
			}
			if len(discovery.PodMatches) > 0 {
				items = append(items, "Closest Kubernetes pod names: "+strings.Join(discovery.PodMatches, ", ")+".")
			}
		}
		if discovery.RequestedNamespace != "" && len(discovery.PodNamespaces) > 0 && !stringSliceContains(discovery.PodNamespaces, discovery.RequestedNamespace) {
			items = append(items, "Matching pod names were discovered in other namespaces: "+strings.Join(discovery.PodNamespaces, ", ")+".")
		}
	}
	return items
}

func discoveryFollowUpSteps(discovery targetDiscovery) []string {
	var steps []string
	if !discovery.ServiceFound && len(discovery.ServiceMatches) > 0 {
		steps = append(steps, "Discovery follow-up: Try `systemctl status "+discovery.ServiceMatches[0]+" --no-pager -l` if that unit name looks like the intended target.")
	}
	if !discovery.RuntimeFound && len(discovery.RuntimeMatches) > 0 {
		runtime := discovery.RuntimeTool
		if runtime == "" {
			runtime = "podman"
		}
		steps = append(steps, "Discovery follow-up: Try `"+runtime+" logs --tail 100 "+discovery.RuntimeMatches[0]+"` or inspect that container if it is the intended target.")
	}
	if !discovery.PodFound && len(discovery.PodMatches) > 0 {
		command := "kubectl describe pod " + discovery.PodMatches[0]
		if discovery.RequestedNamespace != "" {
			command = "kubectl describe pod -n " + discovery.RequestedNamespace + " " + discovery.PodMatches[0]
		}
		steps = append(steps, "Discovery follow-up: Try `"+command+"` if that pod name matches the workload you meant.")
	}
	return steps
}

func applyTargetDiscovery(hypotheses []troubleshootHypothesis, discovery targetDiscovery) {
	for i := range hypotheses {
		switch hypothesisLabel(hypotheses[i].Candidate) {
		case "Service hypothesis":
			if discovery.ServiceFound {
				hypotheses[i].Score += 0.95
				hypotheses[i].Reason += "; target discovery found a matching systemd unit"
			} else if discovery.RuntimeFound || discovery.PodFound {
				hypotheses[i].Score -= 0.35
			}
		case "Container hypothesis":
			if discovery.RuntimeHintRequested && discovery.RuntimeToolAvailable {
				hypotheses[i].Score += 0.20
				hypotheses[i].Reason += "; runtime hint is available on this host"
			}
			if discovery.RuntimeFound {
				hypotheses[i].Score += 0.95
				hypotheses[i].Reason += "; target discovery found a matching container name"
			} else if discovery.ServiceFound || discovery.PodFound {
				hypotheses[i].Score -= 0.25
			}
		case "Kubernetes hypothesis":
			if discovery.PodFound {
				boost := 1.0
				reason := "; target discovery found a matching pod name"
				if discovery.RequestedNamespace != "" && discovery.PodFoundInNamespace {
					boost = 1.20
					reason = "; target discovery found a matching pod name in the requested namespace"
				}
				hypotheses[i].Score += boost
				hypotheses[i].Reason += reason
			} else if discovery.RequestedNamespace != "" && len(discovery.PodNamespaces) > 0 {
				hypotheses[i].Score -= 0.15
				hypotheses[i].Reason += "; matching pod names were only found in different namespaces"
			} else if discovery.ServiceFound || discovery.RuntimeFound {
				hypotheses[i].Score -= 0.25
			}
		}
		if hypotheses[i].Score < 0.05 {
			hypotheses[i].Score = 0.05
		}
	}
}

func liveDiscoverTargetKindsForQuery(query string, env models.Environment, collector evidence.Collector) targetDiscovery {
	target := inferTroubleshootTarget(query)
	if !safeDiscoveryTarget(target) {
		return targetDiscovery{}
	}
	discovery := targetDiscovery{
		RuntimeTool:          requestedRuntimeTool(query, env, collector),
		RuntimeHintRequested: mentionsExplicitRuntime(query),
		RequestedNamespace:   extractNamespace(query),
	}
	variants := targetVariants(target)

	if toolAvailable(env, collector, "systemctl") {
		lines := collector.RunLinesForDetection("systemctl list-units --type=service --all --no-legend --no-pager 2>/dev/null | awk '{print $1}' | sed 's/\\.service$//'")
		discovery.ServiceFound = lineMatchesAny(lines, variants)
		discovery.ServiceMatches = closeMatches(lines, variants, 3)
	}

	runtime := discovery.RuntimeTool
	discovery.RuntimeToolAvailable = runtime != "" && toolAvailable(env, collector, runtime)
	if runtime == "podman" || runtime == "docker" {
		lines := collector.RunLinesForDetection(runtime + " ps -a --format '{{.Names}}' 2>/dev/null")
		discovery.RuntimeFound = lineMatchesAny(lines, variants)
		discovery.RuntimeMatches = closeMatches(lines, variants, 3)
	}

	if toolAvailable(env, collector, "kubectl") {
		entries := collector.RunLinesForDetection("kubectl get pods -A --no-headers 2>/dev/null | awk '{print $1\"/\"$2}'")
		discovery.PodFound, discovery.PodFoundInNamespace = podEntriesMatch(entries, variants, discovery.RequestedNamespace)
		discovery.PodMatches, discovery.PodNamespaces = closePodMatches(entries, variants, discovery.RequestedNamespace, 3)
	}

	return discovery
}

func requestedRuntimeTool(query string, env models.Environment, collector evidence.Collector) string {
	normalized := strings.ToLower(strings.TrimSpace(query))
	switch {
	case strings.Contains(normalized, "podman"):
		return "podman"
	case strings.Contains(normalized, "docker"):
		return "docker"
	default:
		return runtimeTool(env, collector)
	}
}

func mentionsExplicitRuntime(query string) bool {
	normalized := strings.ToLower(strings.TrimSpace(query))
	return strings.Contains(normalized, "podman") || strings.Contains(normalized, "docker")
}

func safeDiscoveryTarget(target string) bool {
	target = strings.TrimSpace(strings.ToLower(target))
	if target == "" {
		return false
	}
	return isLikelyNameToken(target)
}

func targetVariants(target string) []string {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return nil
	}
	variants := map[string]bool{target: true}
	if parts := splitAlphaNumericBoundary(target); len(parts) == 2 {
		variants[parts[0]+"-"+parts[1]] = true
	}
	out := make([]string, 0, len(variants))
	for variant := range variants {
		out = append(out, variant)
	}
	sort.Strings(out)
	return out
}

func splitAlphaNumericBoundary(value string) []string {
	for i := 1; i < len(value); i++ {
		if isDigit(value[i]) && isLowerLetter(value[i-1]) {
			return []string{value[:i], value[i:]}
		}
	}
	return nil
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isLowerLetter(b byte) bool {
	return b >= 'a' && b <= 'z'
}

func lineMatchesAny(lines []string, variants []string) bool {
	for _, line := range lines {
		normalized := strings.ToLower(strings.TrimSpace(line))
		for _, variant := range variants {
			if normalized == variant {
				return true
			}
		}
	}
	return false
}

func closeMatches(lines []string, variants []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	type candidate struct {
		name  string
		score int
	}
	seen := map[string]bool{}
	var ranked []candidate
	for _, line := range lines {
		name := strings.ToLower(strings.TrimSpace(line))
		if name == "" || seen[name] || matchesAnyVariant(name, variants) {
			continue
		}
		score := matchScore(name, variants)
		if score <= 0 {
			continue
		}
		seen[name] = true
		ranked = append(ranked, candidate{name: name, score: score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].name < ranked[j].name
		}
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]string, 0, len(ranked))
	for _, item := range ranked {
		out = append(out, item.name)
	}
	return out
}

func podEntriesMatch(entries []string, variants []string, requestedNamespace string) (bool, bool) {
	found := false
	foundInNamespace := false
	for _, entry := range entries {
		namespace, name, ok := splitNamespacedEntry(entry)
		if !ok {
			continue
		}
		if matchesAnyVariant(strings.ToLower(strings.TrimSpace(name)), variants) {
			found = true
			if requestedNamespace == "" || namespace == requestedNamespace {
				foundInNamespace = true
			}
		}
	}
	return found, foundInNamespace
}

func closePodMatches(entries []string, variants []string, requestedNamespace string, limit int) ([]string, []string) {
	type candidate struct {
		name      string
		namespace string
		score     int
	}
	if limit <= 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	namespaceSeen := map[string]bool{}
	var namespaces []string
	var ranked []candidate
	for _, entry := range entries {
		namespace, name, ok := splitNamespacedEntry(entry)
		if !ok {
			continue
		}
		normalizedName := strings.ToLower(strings.TrimSpace(name))
		if normalizedName == "" || matchesAnyVariant(normalizedName, variants) {
			if requestedNamespace != "" && namespace != "" && namespace != requestedNamespace && matchesAnyVariant(normalizedName, variants) && !namespaceSeen[namespace] {
				namespaces = append(namespaces, namespace)
				namespaceSeen[namespace] = true
			}
			continue
		}
		key := namespace + "/" + normalizedName
		if seen[key] {
			continue
		}
		score := matchScore(normalizedName, variants)
		if score <= 0 {
			continue
		}
		if requestedNamespace != "" && namespace == requestedNamespace {
			score += 10
		}
		seen[key] = true
		ranked = append(ranked, candidate{name: normalizedName, namespace: namespace, score: score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			if ranked[i].name == ranked[j].name {
				return ranked[i].namespace < ranked[j].namespace
			}
			return ranked[i].name < ranked[j].name
		}
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]string, 0, len(ranked))
	for _, item := range ranked {
		out = append(out, item.name)
	}
	sort.Strings(namespaces)
	return out, namespaces
}

func splitNamespacedEntry(entry string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(strings.ToLower(entry)), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func matchesAnyVariant(name string, variants []string) bool {
	for _, variant := range variants {
		if name == variant {
			return true
		}
	}
	return false
}

func matchScore(name string, variants []string) int {
	normalizedName := normalizeInventoryName(name)
	best := 0
	for _, variant := range variants {
		normalizedVariant := normalizeInventoryName(variant)
		switch {
		case normalizedName == normalizedVariant:
			return 100
		case strings.Contains(normalizedName, normalizedVariant):
			score := 80 - (len(normalizedName) - len(normalizedVariant))
			if score > best {
				best = score
			}
		case strings.Contains(normalizedVariant, normalizedName):
			score := 60 - (len(normalizedVariant) - len(normalizedName))
			if score > best {
				best = score
			}
		case sharedPrefixLen(normalizedName, normalizedVariant) >= 4:
			score := 40 + sharedPrefixLen(normalizedName, normalizedVariant)
			if score > best {
				best = score
			}
		}
	}
	return best
}

func normalizeInventoryName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("-", "", "_", "", ".", "", "@", "", ":", "")
	return replacer.Replace(value)
}

func sharedPrefixLen(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

func hypothesesFromCandidates(candidates []models.LLMIntentCandidate) []troubleshootHypothesis {
	var hypotheses []troubleshootHypothesis
	for _, candidate := range candidates {
		if candidate.Intent == "" {
			continue
		}
		reason := candidate.Reasoning
		if strings.TrimSpace(reason) == "" {
			reason = "local LLM fallback ranked this as a plausible interpretation of the outage question"
		}
		hypotheses = append(hypotheses, troubleshootHypothesis{
			Candidate: candidate,
			Score:     candidate.Confidence,
			Reason:    reason,
		})
	}
	return hypotheses
}

func formatHypothesisSteps(hypotheses []troubleshootHypothesis, responses []models.Response) []string {
	steps := make([]string, 0, len(hypotheses))
	for i, hypothesis := range hypotheses {
		response := responses[i]
		steps = append(steps, fmt.Sprintf("%d. %s (%.2f): run `%s`. Why: %s", i+1, hypothesisLabel(hypothesis.Candidate), hypothesis.Score, response.Command, hypothesis.Reason))
	}
	return steps
}

func hypothesisLabel(candidate models.LLMIntentCandidate) string {
	switch candidate.Intent {
	case "service_troubleshoot", "service_status", "service_logs":
		return "Service hypothesis"
	case "runtime_troubleshoot", "runtime_logs", "runtime_inspect":
		return "Container hypothesis"
	case "k8s_pod_troubleshoot", "k8s_pod_status", "k8s_pod_logs":
		return "Kubernetes hypothesis"
	default:
		return "Fallback hypothesis"
	}
}

func troubleshootExplanation(query string, top troubleshootHypothesis, count int) string {
	if count == 1 {
		return fmt.Sprintf("Built a read-only troubleshoot plan for %q and selected the safest high-signal first probe.", query)
	}
	return fmt.Sprintf("Built a ranked, read-only troubleshoot plan for %q instead of guessing a single object type. Start with the highest-confidence probe and branch only if the target turns out to be a different resource.", query)
}

func planConfidence(score float64, count int) string {
	switch {
	case score >= 0.85 && count == 1:
		return "High"
	case score >= 0.70:
		return "Medium"
	default:
		return "Low"
	}
}

func planVerificationSources(query string, hypotheses []troubleshootHypothesis, env models.Environment) []string {
	sources := []string{"query analysis"}
	seen := map[string]bool{"query analysis": true}
	for _, hypothesis := range hypotheses {
		intent := hypothesis.Candidate.Intent
		if !seen[intent] {
			sources = append(sources, intent)
			seen[intent] = true
		}
	}
	for _, name := range []string{"systemctl", "kubectl", "docker", "podman", "containerd"} {
		if env.Commands[name].Exists && !seen["env:"+name] {
			sources = append(sources, "env:"+name)
			seen["env:"+name] = true
		}
	}
	return sources
}

func looksLikeTroubleshootQuery(normalized string) bool {
	for _, marker := range []string{
		"why is", "why won't", "why wont", "not up", "down", "not starting", "failed to start", "not running",
		"unhealthy", "crashloop", "imagepull", "pending", "connection refused", "cannot connect", "can't connect", "no space left", "disk full",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func mentionsService(normalized string) bool {
	return strings.Contains(normalized, "service") || strings.Contains(normalized, "systemd") || strings.Contains(normalized, "unit")
}

func mentionsContainerRuntime(normalized string) bool {
	for _, marker := range []string{"container", "docker", "podman", "containerd", "nerdctl", "crictl", "ctr"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func mentionsKubernetes(normalized string) bool {
	for _, marker := range []string{"kubernetes", "kubectl", "k8s", "pod ", " pod", "deployment", "namespace", "crashloop", "imagepull", "pending pod"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func runtimeTool(env models.Environment, collector evidence.Collector) string {
	for _, name := range []string{"podman", "docker", "nerdctl", "crictl", "ctr", "containerd"} {
		if toolAvailable(env, collector, name) {
			return name
		}
	}
	return ""
}

func toolAvailable(env models.Environment, collector evidence.Collector, name string) bool {
	if env.Commands[name].Exists {
		return true
	}
	return collector.Lookup(name).Exists
}

func inferTroubleshootTarget(query string) string {
	normalized := strings.ToLower(strings.TrimSpace(query))
	cleaner := strings.NewReplacer("?", " ", "!", " ", ",", " ", "\"", " ", "'", " ")
	tokens := strings.Fields(cleaner.Replace(normalized))
	if len(tokens) == 0 {
		return "service"
	}

	stopwords := map[string]bool{
		"why": true, "is": true, "are": true, "the": true, "a": true, "an": true, "not": true, "up": true, "down": true,
		"starting": true, "running": true, "worker": false, "service": true, "pod": true, "container": true, "in": true,
		"on": true, "of": true, "for": true, "to": true, "my": true, "our": true, "node": false,
	}

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		if stopwords[token] {
			continue
		}
		if token == "worker" || token == "node" {
			if i+1 < len(tokens) && isNumericToken(tokens[i+1]) {
				return token + tokens[i+1]
			}
			return token
		}
		if isLikelyNameToken(token) {
			if i+1 < len(tokens) && isNumericToken(tokens[i+1]) {
				return token + tokens[i+1]
			}
			return token
		}
	}
	return "service"
}

func isLikelyNameToken(token string) bool {
	if token == "" {
		return false
	}
	for _, marker := range []string{"why", "not", "up", "down", "starting", "running", "failed"} {
		if token == marker {
			return false
		}
	}
	for _, r := range token {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func isNumericToken(token string) bool {
	_, err := strconv.Atoi(token)
	return err == nil
}
