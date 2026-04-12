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
	response := models.Response{
		IntentID:       "troubleshoot_plan",
		Command:        top.Command,
		Explanation:    troubleshootExplanation(query, topHypothesis, len(kept)),
		ExpectedOutput: "The first probe should confirm the object type and surface the immediate failure reason before any restart, rollout, or package change.",
		Risk:           top.Risk,
		Confidence:     planConfidence(topHypothesis.Score, len(kept)),
		VerifiedFrom:   append([]string{}, top.VerifiedFrom...),
	}

	response.VerifiedFrom = append(response.VerifiedFrom, planVerificationSources(query, kept, env)...)
	response.NextSteps = append(response.NextSteps, formatHypothesisSteps(kept, resolved)...)
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
		if strings.Contains(target, " ") {
			k8sTarget = strings.ReplaceAll(target, " ", "-")
		}
		score := 0.55
		reason := "kubectl is available and the target could be a pod or workload name"
		if mentionsKubernetes(normalized) {
			score = 0.86
			reason = "the query explicitly mentions Kubernetes or pod semantics"
		}
		appendUnique(models.LLMIntentCandidate{Intent: "k8s_pod_troubleshoot", Target: k8sTarget}, score, reason)
		appendUnique(models.LLMIntentCandidate{Intent: "k8s_pod_logs", Target: k8sTarget}, score-0.09, "pod logs are usually the next confirming signal after describe output")
	}

	return hypotheses
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
