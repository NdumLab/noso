package registry

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/pkg/models"
)

func TestTroubleshootPlanDiscoveryPrefersPod(t *testing.T) {
	original := discoverTargetKinds
	discoverTargetKinds = func(query string, env models.Environment, collector evidence.Collector) targetDiscovery {
		return targetDiscovery{PodFound: true, PodMatches: []string{"worker-2"}}
	}
	defer func() { discoverTargetKinds = original }()

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
	if response.Command != "kubectl describe pod worker2" {
		t.Fatalf("Command = %q", response.Command)
	}
	if !containsString(response.Discovery, "Found matching Kubernetes pod name for worker2.") {
		t.Fatalf("Discovery = %#v", response.Discovery)
	}
	if !strings.Contains(strings.Join(response.NextSteps, "\n"), "matching pod name") {
		t.Fatalf("NextSteps = %#v", response.NextSteps)
	}
}

func TestTargetVariantsAddsDashedForm(t *testing.T) {
	variants := targetVariants("worker2")
	if !containsString(variants, "worker2") || !containsString(variants, "worker-2") {
		t.Fatalf("variants = %#v", variants)
	}
}

func TestCloseMatchesIncludesNearbyNames(t *testing.T) {
	matches := closeMatches([]string{"worker@2", "worker2-api", "db", "api-worker"}, targetVariants("worker2"), 3)
	if !containsString(matches, "worker@2") {
		t.Fatalf("matches = %#v", matches)
	}
	if !containsString(matches, "worker2-api") {
		t.Fatalf("matches = %#v", matches)
	}
}

func TestFormatDiscoveryEvidenceIncludesClosestMatches(t *testing.T) {
	env := models.Environment{
		Commands: map[string]models.CommandInfo{
			"systemctl": {Exists: true, Path: "/usr/bin/systemctl"},
			"podman":    {Exists: true, Path: "/usr/bin/podman"},
			"kubectl":   {Exists: true, Path: "/usr/bin/kubectl"},
		},
	}
	items := formatDiscoveryEvidence(targetDiscovery{
		RuntimeTool:    "podman",
		ServiceMatches: []string{"worker@2", "worker2-api"},
		RuntimeMatches: []string{"worker-2"},
		PodMatches:     []string{"worker-2"},
	}, "worker2", env, evidence.NewCollector())
	if !containsString(items, "Closest systemd unit names: worker@2, worker2-api.") {
		t.Fatalf("items = %#v", items)
	}
	if !containsString(items, "Closest podman container names: worker-2.") {
		t.Fatalf("items = %#v", items)
	}
	if !containsString(items, "Closest Kubernetes pod names: worker-2.") {
		t.Fatalf("items = %#v", items)
	}
}

func TestDiscoveryFollowUpStepsSuggestCorrectedTargets(t *testing.T) {
	steps := discoveryFollowUpSteps(targetDiscovery{
		RuntimeTool:    "podman",
		ServiceMatches: []string{"worker@2"},
		RuntimeMatches: []string{"worker2-api"},
		PodMatches:     []string{"worker-2"},
	})
	if !strings.Contains(strings.Join(steps, "\n"), "systemctl status worker@2 --no-pager -l") {
		t.Fatalf("steps = %#v", steps)
	}
	if !strings.Contains(strings.Join(steps, "\n"), "podman logs --tail 100 worker2-api") {
		t.Fatalf("steps = %#v", steps)
	}
	if !strings.Contains(strings.Join(steps, "\n"), "kubectl describe pod worker-2") {
		t.Fatalf("steps = %#v", steps)
	}
}

func TestTroubleshootPlanAmbiguousWorker(t *testing.T) {
	original := discoverTargetKinds
	discoverTargetKinds = func(query string, env models.Environment, collector evidence.Collector) targetDiscovery {
		return targetDiscovery{}
	}
	defer func() { discoverTargetKinds = original }()

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

func TestFormatDiscoveryEvidenceIncludesNamespaceAndRuntimeHint(t *testing.T) {
	env := models.Environment{
		Commands: map[string]models.CommandInfo{
			"systemctl": {Exists: true, Path: "/usr/bin/systemctl"},
			"podman":    {Exists: true, Path: "/usr/bin/podman"},
			"kubectl":   {Exists: true, Path: "/usr/bin/kubectl"},
		},
	}
	items := formatDiscoveryEvidence(targetDiscovery{
		RuntimeTool:          "podman",
		RuntimeHintRequested: true,
		RuntimeToolAvailable: true,
		RequestedNamespace:   "prod",
		PodFound:             true,
		PodFoundInNamespace:  true,
	}, "worker-2", env, evidence.NewCollector())
	if !containsString(items, "Runtime hint confirmed: podman is available on this host.") {
		t.Fatalf("items = %#v", items)
	}
	if !containsString(items, "Found matching Kubernetes pod name for worker-2 in namespace prod.") {
		t.Fatalf("items = %#v", items)
	}
}

func TestDiscoveryFollowUpStepsUseRequestedNamespaceAndRuntime(t *testing.T) {
	steps := discoveryFollowUpSteps(targetDiscovery{
		RuntimeTool:        "docker",
		RequestedNamespace: "prod",
		RuntimeMatches:     []string{"worker2-api"},
		PodMatches:         []string{"worker-2"},
	})
	if !strings.Contains(strings.Join(steps, "\n"), "docker logs --tail 100 worker2-api") {
		t.Fatalf("steps = %#v", steps)
	}
	if !strings.Contains(strings.Join(steps, "\n"), "kubectl describe pod -n prod worker-2") {
		t.Fatalf("steps = %#v", steps)
	}
}

func TestPodEntriesMatchRespectsRequestedNamespace(t *testing.T) {
	found, foundInNamespace := podEntriesMatch([]string{"dev/worker-2", "prod/api"}, targetVariants("worker2"), "prod")
	if !found {
		t.Fatal("found = false, want true")
	}
	if foundInNamespace {
		t.Fatal("foundInNamespace = true, want false")
	}
}

func TestClosePodMatchesReportsOtherNamespaces(t *testing.T) {
	matches, namespaces := closePodMatches([]string{"dev/worker-2", "prod/worker-3", "qa/worker-2"}, targetVariants("worker2"), "prod", 3)
	if !containsString(matches, "worker-3") {
		t.Fatalf("matches = %#v", matches)
	}
	if !containsString(namespaces, "dev") || !containsString(namespaces, "qa") {
		t.Fatalf("namespaces = %#v", namespaces)
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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
