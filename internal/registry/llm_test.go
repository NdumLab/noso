package registry

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/pkg/models"
)

func TestResolveLLMCandidateServiceStatus(t *testing.T) {
	resp, ok, err := ResolveLLMCandidate(models.LLMIntentCandidate{
		Intent:     "service_status",
		Target:     "worker2",
		Confidence: 0.9,
	}, models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("ResolveLLMCandidate() error = %v", err)
	}
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if !strings.Contains(resp.Command, "systemctl status worker2 --no-pager -l") {
		t.Fatalf("Command = %q", resp.Command)
	}
}

func TestClarificationResponse(t *testing.T) {
	resp := ClarificationResponse("Clarify this.", []models.LLMIntentCandidate{{
		Intent:     "service_troubleshoot",
		Target:     "worker2",
		Confidence: 0.68,
	}})
	if resp.IntentID != "clarify_query" {
		t.Fatalf("IntentID = %q", resp.IntentID)
	}
	if len(resp.Warnings) == 0 {
		t.Fatal("Warnings should include candidate summary")
	}
}
