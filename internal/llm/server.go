package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/NdumLab/noso/pkg/models"
)

var numberedWorkerRegex = regexp.MustCompile(`\b([a-zA-Z][a-zA-Z0-9_-]*)\s+(\d+)\b`)
var whyIsTargetRegex = regexp.MustCompile(`\bwhy\s+is\s+(.+?)\s+(?:not\s+up|down|not\s+running|not\s+starting|failed)\b`)

type Server struct {
	provider Provider
	metrics  *Metrics
	logger   *RequestLogger
}

func NewHandler() http.Handler {
	return NewHandlerWithOptions(NewHeuristicProvider(), nil, nil)
}

func NewHandlerWithProvider(provider Provider) http.Handler {
	return NewHandlerWithOptions(provider, nil, nil)
}

func NewHandlerWithOptions(provider Provider, metrics *Metrics, logger *RequestLogger) http.Handler {
	server := Server{provider: provider, metrics: metrics, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.healthHandler)
	mux.HandleFunc("/metrics", server.metricsHandler)
	mux.HandleFunc("/v1/interpret", server.interpretHandler)
	return mux
}

func (s Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"provider": s.provider.Name(),
		"model":    s.provider.Model(),
	})
}

func (s Server) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.metrics.Snapshot(s.provider.Name(), s.provider.Model()))
}

func (s Server) interpretHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LLMInterpretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	resp, err := s.provider.Interpret(context.Background(), req)
	if err != nil {
		s.metrics.RecordError(err)
		_ = s.logger.Append(s.provider.Name(), s.provider.Model(), req.Query, models.LLMInterpretResponse{}, err)
		http.Error(w, fmt.Sprintf("provider error: %v", err), http.StatusBadGateway)
		return
	}
	validated, err := ValidateResponse(resp, req)
	if err != nil {
		s.metrics.RecordError(classifyDecodeError("server validate", err))
		_ = s.logger.Append(s.provider.Name(), s.provider.Model(), req.Query, models.LLMInterpretResponse{}, err)
		http.Error(w, fmt.Sprintf("provider returned invalid response: %v", err), http.StatusBadGateway)
		return
	}
	s.metrics.RecordRequest(validated)
	_ = s.logger.Append(s.provider.Name(), s.provider.Model(), req.Query, validated, nil)
	writeJSON(w, http.StatusOK, validated)
}

func interpret(req models.LLMInterpretRequest) models.LLMInterpretResponse {
	query := strings.TrimSpace(strings.ToLower(req.Query))
	if query == "" {
		return models.LLMInterpretResponse{Status: "ok"}
	}

	if resp, ok := explicitCandidate(query, req.Environment.AvailableTools); ok {
		return resp
	}

	target := normalizeTarget(extractTarget(query))
	ambiguous := strings.Contains(query, "not up") || strings.Contains(query, "down") || strings.Contains(query, "not running")
	if ambiguous && target != "" {
		candidates := ambiguousCandidates(target, req.Environment.AvailableTools)
		if len(candidates) > 1 {
			return models.LLMInterpretResponse{
				Status:                "ok",
				NeedsClarification:    true,
				ClarificationQuestion: "Do you mean a systemd service, a container, or a Kubernetes pod?",
				Candidates:            trimCandidates(candidates, req.Hints.MaxCandidates),
			}
		}
		if len(candidates) == 1 {
			return models.LLMInterpretResponse{
				Status:     "ok",
				Candidates: candidates,
			}
		}
	}

	return models.LLMInterpretResponse{Status: "ok"}
}

func explicitCandidate(query string, tools []string) (models.LLMInterpretResponse, bool) {
	switch {
	case strings.Contains(query, "git push"):
		return single("git_push", "", "", "git", 0.98, "The query explicitly asks about git push."), true
	case strings.Contains(query, "install "):
		target := normalizeTarget(extractAfter(query, "install "))
		if target == "" {
			return models.LLMInterpretResponse{}, false
		}
		return single("package_install", target, "", "", 0.96, "The query explicitly requests package installation guidance."), true
	case strings.Contains(query, "nslookup ") || strings.Contains(query, "dig ") || strings.Contains(query, "dns lookup"):
		target := normalizeTarget(extractDNSHost(query))
		if target == "" {
			return models.LLMInterpretResponse{}, false
		}
		return single("dns_lookup", target, "", "", 0.97, "The query explicitly asks for DNS lookup guidance."), true
	case strings.Contains(query, "cron jobs") || strings.Contains(query, "crontab -l"):
		return single("cron_list", "", "", "", 0.95, "The query explicitly asks to list cron entries."), true
	case strings.Contains(query, "logs") && strings.Contains(query, "service"):
		target := normalizeTarget(extractBefore(query, " service"))
		if target == "" {
			target = "service"
		}
		return single("service_logs", target, "", "systemctl", 0.92, "The query explicitly asks for service logs."), true
	case strings.Contains(query, "service") && (strings.Contains(query, "status") || strings.Contains(query, "not starting") || strings.Contains(query, "failed")):
		target := normalizeTarget(extractBefore(query, " service"))
		intent := "service_status"
		confidence := 0.91
		reason := "The query explicitly mentions a service status check."
		if strings.Contains(query, "not starting") || strings.Contains(query, "failed") {
			intent = "service_troubleshoot"
			confidence = 0.88
			reason = "The query explicitly mentions a failing system service."
		}
		return single(intent, target, "", "systemctl", confidence, reason), true
	case strings.Contains(query, "pod") && strings.Contains(query, "logs"):
		target := normalizeTarget(extractAfter(query, "pod "))
		return single("k8s_pod_logs", target, "", "kubectl", 0.91, "The query explicitly asks for Kubernetes pod logs."), true
	case strings.Contains(query, "pod") && (strings.Contains(query, "not running") || strings.Contains(query, "failed") || strings.Contains(query, "status")):
		target := normalizeTarget(extractAfter(query, "pod "))
		intent := "k8s_pod_status"
		if strings.Contains(query, "not running") || strings.Contains(query, "failed") {
			intent = "k8s_pod_troubleshoot"
		}
		return single(intent, target, "", "kubectl", 0.89, "The query explicitly mentions a Kubernetes pod."), true
	case (strings.Contains(query, "container") || strings.Contains(query, "docker") || strings.Contains(query, "podman")) && strings.Contains(query, "logs"):
		target := normalizeTarget(extractContainerTarget(query))
		return single("runtime_logs", target, "", preferredRuntime(tools, query), 0.9, "The query explicitly asks for container logs."), true
	case (strings.Contains(query, "container") || strings.Contains(query, "docker") || strings.Contains(query, "podman")) &&
		(strings.Contains(query, "not running") || strings.Contains(query, "failed") || strings.Contains(query, "not starting")):
		target := normalizeTarget(extractContainerTarget(query))
		return single("runtime_troubleshoot", target, "", preferredRuntime(tools, query), 0.87, "The query explicitly mentions a failing container."), true
	default:
		return models.LLMInterpretResponse{}, false
	}
}

func ambiguousCandidates(target string, tools []string) []models.LLMIntentCandidate {
	var candidates []models.LLMIntentCandidate
	if hasTool(tools, "systemctl") {
		candidates = append(candidates, models.LLMIntentCandidate{
			Intent:     "service_troubleshoot",
			Target:     target,
			ToolHint:   "systemctl",
			Confidence: 0.68,
			Reasoning:  "The wording matches a service that is down or failed to start.",
		})
	}
	if hasTool(tools, "kubectl") {
		candidates = append(candidates, models.LLMIntentCandidate{
			Intent:     "k8s_pod_troubleshoot",
			Target:     hyphenateNumberedTarget(target),
			ToolHint:   "kubectl",
			Confidence: 0.53,
			Reasoning:  "Worker-style names are also common for pods or replicas.",
		})
	}
	runtime := preferredRuntime(tools, "")
	if runtime != "" {
		candidates = append(candidates, models.LLMIntentCandidate{
			Intent:     "runtime_troubleshoot",
			Target:     hyphenateNumberedTarget(target),
			ToolHint:   runtime,
			Confidence: 0.47,
			Reasoning:  "The target could also refer to a container instance.",
		})
	}
	return candidates
}

func preferredRuntime(tools []string, query string) string {
	switch {
	case strings.Contains(query, "podman"):
		return "podman"
	case strings.Contains(query, "docker"):
		return "docker"
	case hasTool(tools, "podman"):
		return "podman"
	case hasTool(tools, "docker"):
		return "docker"
	default:
		return ""
	}
}

func hasTool(tools []string, want string) bool {
	for _, tool := range tools {
		if tool == want {
			return true
		}
	}
	return false
}

func extractTarget(query string) string {
	if matches := whyIsTargetRegex.FindStringSubmatch(query); len(matches) == 2 {
		return matches[1]
	}
	fields := strings.Fields(query)
	for idx, field := range fields {
		if field == "is" || field == "not" || field == "down" {
			if idx > 0 {
				return strings.Join(fields[:idx], " ")
			}
			break
		}
	}
	return query
}

func extractAfter(query, marker string) string {
	idx := strings.Index(query, marker)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(query[idx+len(marker):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func extractBefore(query, marker string) string {
	idx := strings.Index(query, marker)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(query[:idx])
}

func extractContainerTarget(query string) string {
	for _, marker := range []string{"container ", "docker ", "podman "} {
		if target := extractAfter(query, marker); target != "" {
			return target
		}
	}
	return ""
}

func extractDNSHost(query string) string {
	for _, marker := range []string{"nslookup ", "dig ", "lookup ", "dns lookup "} {
		if target := extractAfter(query, marker); target != "" {
			return target
		}
	}
	return ""
}

func normalizeTarget(raw string) string {
	raw = strings.TrimSpace(strings.Trim(raw, "`\"'.,"))
	if raw == "" {
		return ""
	}
	if matches := numberedWorkerRegex.FindStringSubmatch(raw); len(matches) == 3 {
		return matches[1] + matches[2]
	}
	return strings.ReplaceAll(raw, " ", "")
}

func hyphenateNumberedTarget(target string) string {
	if matches := regexp.MustCompile(`^([a-zA-Z][a-zA-Z_-]*?)(\d+)$`).FindStringSubmatch(target); len(matches) == 3 {
		return matches[1] + "-" + matches[2]
	}
	return target
}

func trimCandidates(candidates []models.LLMIntentCandidate, limit int) []models.LLMIntentCandidate {
	if limit <= 0 || len(candidates) <= limit {
		return candidates
	}
	return candidates[:limit]
}

func single(intent, target, namespace, toolHint string, confidence float64, reason string) models.LLMInterpretResponse {
	return models.LLMInterpretResponse{
		Status: "ok",
		Candidates: []models.LLMIntentCandidate{{
			Intent:     intent,
			Target:     target,
			Namespace:  namespace,
			ToolHint:   toolHint,
			Confidence: confidence,
			Reasoning:  reason,
		}},
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
