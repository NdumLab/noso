package registry

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/pkg/models"
)

func TestTroubleshootPlanAmbiguousWorker(t *testing.T) {
	env := models.Environment{
		Commands: map[string]models.CommandInfo{
			"systemctl": {Exists: true, Path: "/usr/bin/systemctl"},
			"podman":    {Exists: true, Path: "/usr/bin/podman"},
			"kubectl":   {Exists: true, Path: "/usr/bin/kubectl"},
		},
	}

	response, ok, err := TroubleshootPlan("why is worker 2 not up?", env, evidence.NewCollector())
	if err != nil {
		t.Fatalf("TroubleshootPlan() error = %v", err)
	}
	if !ok {
		t.Fatal("expected troubleshoot plan")
	}
	if response.IntentID != "troubleshoot_plan" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Command != "systemctl status worker2 --no-pager -l" {
		t.Fatalf("Command = %q", response.Command)
	}
	if len(response.NextSteps) < 3 {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if !strings.Contains(strings.Join(response.NextSteps, "\n"), "Container hypothesis") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
	if !strings.Contains(strings.Join(response.NextSteps, "\n"), "Kubernetes hypothesis") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestTroubleshootPlanFromCandidates(t *testing.T) {
	env := models.Environment{
		Commands: map[string]models.CommandInfo{
			"systemctl": {Exists: true, Path: "/usr/bin/systemctl"},
			"kubectl":   {Exists: true, Path: "/usr/bin/kubectl"},
		},
	}

	response, ok, err := TroubleshootPlanFromCandidates("why is worker 2 not up?", env, evidence.NewCollector(), []models.LLMIntentCandidate{
		{Intent: "service_troubleshoot", Target: "worker2", Confidence: 0.68, Reasoning: "worker-like names often map to services"},
		{Intent: "k8s_pod_troubleshoot", Target: "worker-2", Confidence: 0.44, Reasoning: "replica names often map to pods"},
	})
	if err != nil {
		t.Fatalf("TroubleshootPlanFromCandidates() error = %v", err)
	}
	if !ok {
		t.Fatal("expected troubleshoot plan from candidates")
	}
	if response.Command != "systemctl status worker2 --no-pager -l" {
		t.Fatalf("Command = %q", response.Command)
	}
	if !strings.Contains(response.Explanation, "ranked, read-only troubleshoot plan") {
		t.Fatalf("Explanation = %q", response.Explanation)
	}
}
