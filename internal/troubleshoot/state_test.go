package troubleshoot

import (
	"path/filepath"
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
		Query:        "why is worker 2 not up?",
		Executed:     []string{"systemctl status worker2 --no-pager -l"},
		LastFindings: []string{"Live service evidence: The requested unit could not be found on this host."},
		FamilyScores: map[string]float64{"service": -2.0, "runtime": 1.0, "kubernetes": 0.7},
	}
	updated := ApplyThreadContext(response, thread)
	if updated.Command != "podman ps -a" {
		t.Fatalf("Command = %q", updated.Command)
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
}

func TestUpdateStatePrependsHistory(t *testing.T) {
	state := UpdateState(State{}, "why is worker 2 not up?", models.Response{
		IntentID: "troubleshoot_plan",
		Command:  "systemctl status worker2 --no-pager -l",
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
