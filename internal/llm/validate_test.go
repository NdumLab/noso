package llm

import (
	"testing"

	"github.com/NdumLab/noso/pkg/models"
)

func TestValidateResponseDropsUnsupportedAndSorts(t *testing.T) {
	resp, err := ValidateResponse(models.LLMInterpretResponse{
		Status: "ok",
		Candidates: []models.LLMIntentCandidate{
			{Intent: "made_up", Confidence: 0.9},
			{Intent: "service_status", Target: "worker2\n", ToolHint: "systemctl", Confidence: 0.6},
			{Intent: "dns_lookup", Target: "example.com", ToolHint: "bogus", Confidence: 2},
		},
	}, models.LLMInterpretRequest{
		Environment: models.LLMEnvironment{
			AvailableTools: []string{"systemctl", "nslookup"},
		},
		Hints: models.LLMInterpretHints{MaxCandidates: 3},
	})
	if err != nil {
		t.Fatalf("ValidateResponse() error = %v", err)
	}
	if len(resp.Candidates) != 2 {
		t.Fatalf("len(Candidates) = %d, want 2", len(resp.Candidates))
	}
	if resp.Candidates[0].Intent != "dns_lookup" {
		t.Fatalf("first candidate intent = %q, want dns_lookup", resp.Candidates[0].Intent)
	}
	if resp.Candidates[0].ToolHint != "" {
		t.Fatalf("ToolHint = %q, want empty after sanitization", resp.Candidates[0].ToolHint)
	}
	if resp.Candidates[1].Target != "worker2" {
		t.Fatalf("Target = %q, want worker2", resp.Candidates[1].Target)
	}
}

func TestValidateResponseRequiresUsableClarificationCandidate(t *testing.T) {
	_, err := ValidateResponse(models.LLMInterpretResponse{
		Status:             "ok",
		NeedsClarification: true,
		Candidates:         []models.LLMIntentCandidate{{Intent: "made_up", Confidence: 0.9}},
	}, models.LLMInterpretRequest{
		Environment: models.LLMEnvironment{},
		Hints:       models.LLMInterpretHints{MaxCandidates: 3},
	})
	if err == nil {
		t.Fatal("ValidateResponse() should fail for clarification with no usable candidates")
	}
}

func TestRankedCandidatesFiltersLowConfidence(t *testing.T) {
	ranked := RankedCandidates(models.LLMInterpretResponse{
		Candidates: []models.LLMIntentCandidate{
			{Intent: "service_status", Confidence: 0.4},
			{Intent: "dns_lookup", Confidence: 0.8},
			{Intent: "git_push", Confidence: 0.6},
		},
	}, 0.5)
	if len(ranked) != 2 {
		t.Fatalf("len(ranked) = %d, want 2", len(ranked))
	}
	if ranked[0].Intent != "dns_lookup" {
		t.Fatalf("ranked[0].Intent = %q, want dns_lookup", ranked[0].Intent)
	}
}
