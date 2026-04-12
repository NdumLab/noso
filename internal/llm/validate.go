package llm

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/NdumLab/noso/pkg/models"
)

var supportedIntents = map[string]bool{
	"service_status":       true,
	"service_logs":         true,
	"service_troubleshoot": true,
	"k8s_pod_status":       true,
	"k8s_pod_logs":         true,
	"k8s_pod_troubleshoot": true,
	"runtime_logs":         true,
	"runtime_inspect":      true,
	"runtime_troubleshoot": true,
	"git_push":             true,
	"package_install":      true,
	"dns_lookup":           true,
	"cron_list":            true,
}

func ValidateResponse(resp models.LLMInterpretResponse, req models.LLMInterpretRequest) (models.LLMInterpretResponse, error) {
	if strings.TrimSpace(resp.Status) == "" {
		resp.Status = "ok"
	}

	allowedTools := map[string]bool{}
	for _, tool := range req.Environment.AvailableTools {
		allowedTools[tool] = true
	}

	sanitized := make([]models.LLMIntentCandidate, 0, len(resp.Candidates))
	for _, candidate := range resp.Candidates {
		candidate.Intent = strings.TrimSpace(candidate.Intent)
		if !supportedIntents[candidate.Intent] {
			continue
		}
		if math.IsNaN(candidate.Confidence) || math.IsInf(candidate.Confidence, 0) || candidate.Confidence <= 0 {
			continue
		}
		if candidate.Confidence > 1 {
			candidate.Confidence = 1
		}
		candidate.Target = sanitizeField(candidate.Target, 96)
		candidate.Namespace = sanitizeField(candidate.Namespace, 96)
		candidate.Reasoning = sanitizeField(candidate.Reasoning, 240)
		candidate.ToolHint = strings.TrimSpace(candidate.ToolHint)
		if candidate.ToolHint != "" && !allowedTools[candidate.ToolHint] {
			candidate.ToolHint = ""
		}
		sanitized = append(sanitized, candidate)
	}

	sort.SliceStable(sanitized, func(i, j int) bool {
		return sanitized[i].Confidence > sanitized[j].Confidence
	})

	maxCandidates := req.Hints.MaxCandidates
	if maxCandidates <= 0 {
		maxCandidates = 3
	}
	if len(sanitized) > maxCandidates {
		sanitized = sanitized[:maxCandidates]
	}

	resp.Candidates = sanitized
	resp.ClarificationQuestion = sanitizeField(resp.ClarificationQuestion, 200)
	if resp.NeedsClarification && resp.ClarificationQuestion == "" {
		resp.ClarificationQuestion = "Do you mean a systemd service, a container, or a Kubernetes pod?"
	}
	if resp.NeedsClarification && len(resp.Candidates) == 0 {
		return models.LLMInterpretResponse{}, fmt.Errorf("llm clarification response contained no usable candidates")
	}
	return resp, nil
}

func sanitizeField(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.Join(strings.Fields(value), " ")
	if maxLen > 0 && len(value) > maxLen {
		value = value[:maxLen]
	}
	return strings.TrimSpace(value)
}

func RankedCandidates(resp models.LLMInterpretResponse, minConfidence float64) []models.LLMIntentCandidate {
	candidates := make([]models.LLMIntentCandidate, 0, len(resp.Candidates))
	for _, candidate := range resp.Candidates {
		if candidate.Confidence >= minConfidence {
			candidates = append(candidates, candidate)
		}
	}
	slices.SortStableFunc(candidates, func(a, b models.LLMIntentCandidate) int {
		switch {
		case a.Confidence > b.Confidence:
			return -1
		case a.Confidence < b.Confidence:
			return 1
		default:
			return 0
		}
	})
	return candidates
}
