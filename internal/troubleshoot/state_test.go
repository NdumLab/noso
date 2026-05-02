package troubleshoot

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/NdumLab/noso/pkg/models"
)

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "troubleshoot-state.json")
	state := UpdateState(State{}, "why is worker 2 not up?", models.Response{
		IntentID: "troubleshoot_plan",
		Command:  "systemctl status worker2 --no-pager -l",
		Findings: []string{"Live service evidence: unit not found"},
		Warnings: []string{"query was ambiguous"},
	})
	if err := SaveState(path, state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}
	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	thread, ok := FindThread(loaded, "why is worker 2 not up?")
	if !ok {
		t.Fatal("FindThread() = false, want true")
	}
	if thread.LastCommand != "systemctl status worker2 --no-pager -l" {
		t.Fatalf("LastCommand = %q", thread.LastCommand)
	}
	if thread.FamilyScores["service"] >= 0 {
		t.Fatalf("FamilyScores = %#v, expected service family to be down-ranked", thread.FamilyScores)
	}
	if thread.CauseScores["service_unit_missing"] <= 0 {
		t.Fatalf("CauseScores = %#v, expected missing-unit cause to be up-ranked", thread.CauseScores)
	}
	if len(thread.LastDiscovery) != 0 {
		t.Fatalf("LastDiscovery = %#v, want none", thread.LastDiscovery)
	}
	if len(thread.History) != 1 {
		t.Fatalf("History len = %d, want 1", len(thread.History))
	}
	if thread.History[0].Summary == "" {
		t.Fatal("expected persisted probe summary")
	}
}

func TestApplyThreadContextAdvancesCommand(t *testing.T) {
	response := models.Response{
		IntentID:    "troubleshoot_plan",
		Command:     "systemctl status worker2 --no-pager -l",
		Explanation: "Built a ranked, read-only troubleshoot plan.",
		NextSteps: []string{
			"1. Service hypothesis (0.82): run `systemctl status worker2 --no-pager -l`.",
			"Evidence follow-up: Run `journalctl -u <service> -n 50 --no-pager` for more detail if the unit is unhealthy.",
		},
	}
	thread := StateThread{
		Query:        "why is worker 2 not up?",
		Executed:     []string{"systemctl status worker2 --no-pager -l"},
		LastFindings: []string{"Live service evidence: unit not found"},
	}
	updated := ApplyThreadContext(response, thread)
	if updated.Command != "journalctl -u <service> -n 50 --no-pager" {
		t.Fatalf("Command = %q", updated.Command)
	}
	if len(updated.Findings) == 0 {
		t.Fatal("expected previous findings to be carried forward")
	}
}

func TestApplyThreadContextPrefersRuntimeAfterMissingService(t *testing.T) {
	response := models.Response{
		IntentID:    "troubleshoot_plan",
		Command:     "systemctl status worker2 --no-pager -l",
		Explanation: "Built a ranked, read-only troubleshoot plan.",
		NextSteps: []string{
			"1. Service hypothesis (0.82): run `systemctl status worker2 --no-pager -l`.",
			"Evidence follow-up: Run `journalctl -u <service> -n 50 --no-pager` for more detail if the unit is unhealthy.",
			"3. Container hypothesis (0.62): run `podman ps -a`.",
			"4. Container hypothesis (0.54): run `podman logs --tail 100 worker2`.",
		},
	}
	thread := StateThread{
		Query:         "why is worker 2 not up?",
		Executed:      []string{"systemctl status worker2 --no-pager -l"},
		LastDiscovery: []string{"No matching systemd unit name found for worker2."},
		LastFindings:  []string{"Live service evidence: The requested unit could not be found on this host."},
		FamilyScores:  map[string]float64{"service": -2.0, "runtime": 1.0, "kubernetes": 0.7},
	}
	updated := ApplyThreadContext(response, thread)
	if updated.Command != "podman ps -a" {
		t.Fatalf("Command = %q", updated.Command)
	}
	if !containsString(updated.Discovery, "Previous discovery: No matching systemd unit name found for worker2.") {
		t.Fatalf("Discovery = %#v", updated.Discovery)
	}
}

func TestApplyThreadContextDoesNotNestHistoricalPrefixes(t *testing.T) {
	response := models.Response{
		IntentID:    "troubleshoot_plan",
		Command:     "systemctl status worker2 --no-pager -l",
		Explanation: "Built a ranked, read-only troubleshoot plan.",
	}
	thread := StateThread{
		Query:        "why is worker 2 not up?",
		Executed:     []string{"systemctl status worker2 --no-pager -l"},
		LastFindings: []string{"Previous finding: Live service evidence: The requested unit could not be found on this host."},
		LastDiscovery: []string{
			"Previous discovery: No matching systemd unit name found for worker2.",
		},
		LastWarnings: []string{"previous thread warning: runtime probe unavailable: podman is not currently installed on this host"},
	}
	updated := ApplyThreadContext(response, thread)
	if !containsString(updated.Findings, "Previous finding: Live service evidence: The requested unit could not be found on this host.") {
		t.Fatalf("Findings = %#v", updated.Findings)
	}
	for _, finding := range updated.Findings {
		if strings.Contains(finding, "Previous finding: Previous finding:") {
			t.Fatalf("Findings = %#v", updated.Findings)
		}
	}
	for _, item := range updated.Discovery {
		if strings.Contains(item, "Previous discovery: Previous discovery:") {
			t.Fatalf("Discovery = %#v", updated.Discovery)
		}
	}
	for _, warning := range updated.Warnings {
		if strings.Contains(strings.ToLower(warning), "previous thread warning: previous thread warning:") {
			t.Fatalf("Warnings = %#v", updated.Warnings)
		}
	}
}

func TestUpdateStateAccumulatesRuntimeConfidence(t *testing.T) {
	state := UpdateState(State{}, "why is worker 2 not up?", models.Response{
		IntentID: "troubleshoot_plan",
		Command:  "podman ps -a",
		Findings: []string{"Runtime evidence: The runtime container list shows non-running or unhealthy containers."},
	})
	thread, ok := FindThread(state, "why is worker 2 not up?")
	if !ok {
		t.Fatal("FindThread() = false, want true")
	}
	if thread.FamilyScores["runtime"] <= 0 {
		t.Fatalf("FamilyScores = %#v, expected runtime family to be up-ranked", thread.FamilyScores)
	}
	if thread.CauseScores["runtime_container_failure"] <= 0 {
		t.Fatalf("CauseScores = %#v, expected runtime cause to be up-ranked", thread.CauseScores)
	}
}

func TestUpdateStatePrependsHistory(t *testing.T) {
	state := UpdateState(State{}, "why is worker 2 not up?", models.Response{
		IntentID: "troubleshoot_plan",
		Command:  "systemctl status worker2 --no-pager -l",
		Discovery: []string{
			"No matching systemd unit name found for worker2.",
		},
		Findings: []string{"Live service evidence: unit not found"},
	})
	state = UpdateState(state, "why is worker 2 not up?", models.Response{
		IntentID: "troubleshoot_plan",
		Command:  "podman ps -a",
		Findings: []string{"Runtime evidence: The runtime container list shows non-running or unhealthy containers."},
	})
	thread, ok := FindThread(state, "why is worker 2 not up?")
	if !ok {
		t.Fatal("FindThread() = false, want true")
	}
	if len(thread.History) != 2 {
		t.Fatalf("History len = %d, want 2", len(thread.History))
	}
	if thread.History[0].Command != "podman ps -a" {
		t.Fatalf("History[0].Command = %q", thread.History[0].Command)
	}
	if thread.History[1].Command != "systemctl status worker2 --no-pager -l" {
		t.Fatalf("History[1].Command = %q", thread.History[1].Command)
	}
	if !containsString(thread.LastDiscovery, "No matching systemd unit name found for worker2.") {
		t.Fatalf("LastDiscovery = %#v", thread.LastDiscovery)
	}
}

func TestResetStateByQuery(t *testing.T) {
	state := UpdateState(State{}, "why is worker 2 not up?", models.Response{
		IntentID: "troubleshoot_plan",
		Command:  "systemctl status worker2 --no-pager -l",
	})
	state = UpdateState(state, "why is api not up?", models.Response{
		IntentID: "troubleshoot_plan",
		Command:  "systemctl status api --no-pager -l",
	})
	state = ResetState(state, "why is worker 2 not up?")
	if _, ok := FindThread(state, "why is worker 2 not up?"); ok {
		t.Fatal("expected worker thread to be removed")
	}
	if _, ok := FindThread(state, "why is api not up?"); !ok {
		t.Fatal("expected api thread to remain")
	}
}

func TestAttachLikelyCausesRanksCauses(t *testing.T) {
	thread := StateThread{
		CauseScores: map[string]float64{
			"dependency_database_connectivity": 2.1,
			"permission_or_access_denied":      1.8,
			"runtime_container_failure":        0.9,
		},
	}
	response := AttachLikelyCauses(models.Response{}, thread)
	if len(response.LikelyCauses) != 3 {
		t.Fatalf("LikelyCauses len = %d, want 3", len(response.LikelyCauses))
	}
	if response.LikelyCauses[0] != "High confidence: the workload is failing because it cannot reach its database dependency" {
		t.Fatalf("LikelyCauses[0] = %q", response.LikelyCauses[0])
	}
	if response.LikelyCauses[1] != "Medium confidence: the workload is failing because of a permission or access-denied error" {
		t.Fatalf("LikelyCauses[1] = %q", response.LikelyCauses[1])
	}
	if len(response.NextSteps) == 0 {
		t.Fatal("expected cause-aware follow-up guidance")
	}
	if response.NextSteps[0] != "Evidence follow-up: Cause follow-up: Verify the database endpoint, credentials source, DNS resolution, and network reachability from the workload host or container." {
		t.Fatalf("NextSteps[0] = %q", response.NextSteps[0])
	}
}

func TestUpdateStateDownranksDisprovenServiceCause(t *testing.T) {
	state := UpdateState(State{}, "why is worker 2 not up?", models.Response{
		IntentID: "troubleshoot_plan",
		Command:  "systemctl status worker2 --no-pager -l",
		Findings: []string{"Live service evidence: The unit is in a failed state. systemd recorded a service failure rather than a healthy running process."},
	})
	state = UpdateState(state, "why is worker 2 not up?", models.Response{
		IntentID: "troubleshoot_plan",
		Command:  "systemctl status worker2 --no-pager -l",
		Findings: []string{"Live service evidence: The requested unit could not be found on this host. That usually means the service name is wrong, the unit file is absent, or the workload is not managed by systemd."},
	})
	thread, ok := FindThread(state, "why is worker 2 not up?")
	if !ok {
		t.Fatal("FindThread() = false, want true")
	}
	if thread.CauseScores["service_unit_missing"] <= 0 {
		t.Fatalf("CauseScores = %#v, expected missing-unit cause to remain", thread.CauseScores)
	}
	if thread.CauseScores["service_process_failure"] != 0 {
		t.Fatalf("CauseScores = %#v, expected service failure cause to be retired", thread.CauseScores)
	}
}

func TestAttachLikelyCausesOmitsRetiredKubernetesCause(t *testing.T) {
	thread := StateThread{
		CauseScores: map[string]float64{
			"kubernetes_crashloop":           2.2,
			"kubernetes_image_pull":          0.4,
			"kubernetes_scheduling_capacity": 0,
		},
	}
	adjustCauseScore(thread.CauseScores, "kubernetes_crashloop", -3.0)
	response := AttachLikelyCauses(models.Response{}, thread)
	for _, cause := range response.LikelyCauses {
		if strings.Contains(strings.ToLower(cause), "crashing repeatedly") {
			t.Fatalf("LikelyCauses = %#v, expected retired cause to be omitted", response.LikelyCauses)
		}
	}
}

func TestPreviewThreadIgnoresPreviousFindingPrefixesForCauseScoring(t *testing.T) {
	existing := StateThread{
		Query:       "why is worker 2 not up?",
		CauseScores: map[string]float64{"service_unit_missing": 0},
	}
	response := models.Response{
		IntentID:    "troubleshoot_plan",
		Explanation: "Built a ranked, read-only troubleshoot plan.",
		Findings:    []string{"Previous finding: Live service evidence: The requested unit could not be found on this host."},
	}
	thread := PreviewThread(existing, existing.Query, response)
	if thread.CauseScores["service_unit_missing"] != 0 {
		t.Fatalf("CauseScores = %#v, expected previous finding prefix to be ignored", thread.CauseScores)
	}
}

func TestPreviewThreadCanonicalizesHistoricalPrefixesInStoredState(t *testing.T) {
	thread := PreviewThread(StateThread{}, "why is worker 2 not up?", models.Response{
		Discovery: []string{"Previous discovery: No matching systemd unit name found for worker2."},
		Findings:  []string{"Previous finding: Live service evidence: The requested unit could not be found on this host."},
		Warnings:  []string{"previous thread warning: runtime probe unavailable: podman is not currently installed on this host"},
	})
	if !containsString(thread.LastDiscovery, "No matching systemd unit name found for worker2.") {
		t.Fatalf("LastDiscovery = %#v", thread.LastDiscovery)
	}
	if !containsString(thread.LastFindings, "Live service evidence: The requested unit could not be found on this host.") {
		t.Fatalf("LastFindings = %#v", thread.LastFindings)
	}
	if !containsString(thread.LastWarnings, "runtime probe unavailable: podman is not currently installed on this host") {
		t.Fatalf("LastWarnings = %#v", thread.LastWarnings)
	}
	for _, item := range append(append([]string{}, thread.LastDiscovery...), append(thread.LastFindings, thread.LastWarnings...)...) {
		lower := strings.ToLower(item)
		if strings.Contains(lower, "previous discovery:") || strings.Contains(lower, "previous finding:") || strings.Contains(lower, "previous thread warning:") {
			t.Fatalf("stored state should be canonicalized, got %#v", thread)
		}
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
