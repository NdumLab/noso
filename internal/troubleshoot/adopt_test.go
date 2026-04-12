package troubleshoot

import "testing"

func TestParseSuggestedTargets(t *testing.T) {
	suggestions := parseSuggestedTargets([]string{
		"Discovery follow-up: Try `systemctl status worker@2 --no-pager -l` if that unit name looks like the intended target.",
		"Discovery follow-up: Try `podman logs --tail 100 worker2-api` or inspect that container if it is the intended target.",
		"Discovery follow-up: Try `kubectl describe pod worker-2` if that pod name matches the workload you meant.",
		"Discovery follow-up: Try `kubectl describe pvc -n prod web-data` if that claim name matches the blocked volume.",
		"Discovery follow-up: Try `kubectl describe deployment -n prod web` if that rollout owns the failing pods.",
		"Discovery follow-up: Try `kubectl describe service -n prod api` if that service matches the failing traffic path.",
	})
	if len(suggestions) != 6 {
		t.Fatalf("len(suggestions) = %d, want 6", len(suggestions))
	}
	if suggestions[0].Family != "service" || suggestions[0].Name != "worker@2" {
		t.Fatalf("suggestions[0] = %#v", suggestions[0])
	}
	if suggestions[1].Family != "runtime" || suggestions[1].Name != "worker2-api" {
		t.Fatalf("suggestions[1] = %#v", suggestions[1])
	}
	if suggestions[2].Family != "kubernetes" || suggestions[2].Name != "worker-2" {
		t.Fatalf("suggestions[2] = %#v", suggestions[2])
	}
	if suggestions[3].Family != "kubernetes-pvc" || suggestions[3].Name != "web-data" || suggestions[3].Namespace != "prod" {
		t.Fatalf("suggestions[3] = %#v", suggestions[3])
	}
	if suggestions[4].Family != "kubernetes-deployment" || suggestions[4].Name != "web" || suggestions[4].Namespace != "prod" {
		t.Fatalf("suggestions[4] = %#v", suggestions[4])
	}
	if suggestions[5].Family != "kubernetes-service" || suggestions[5].Name != "api" || suggestions[5].Namespace != "prod" {
		t.Fatalf("suggestions[5] = %#v", suggestions[5])
	}
}

func TestResolveSuggestedTargetUsesStoredSuggestion(t *testing.T) {
	state := State{
		Threads: []StateThread{{
			Query: "why is worker 2 not up?",
			SuggestedTargets: []SuggestedTarget{{
				Family: "kubernetes",
				Name:   "worker-2",
			}},
		}},
	}
	thread, suggestion, ok := ResolveSuggestedTarget(state, "check worker-2")
	if !ok {
		t.Fatal("ResolveSuggestedTarget() = false, want true")
	}
	if thread.Query != "why is worker 2 not up?" {
		t.Fatalf("thread.Query = %q", thread.Query)
	}
	if suggestion.Name != "worker-2" || suggestion.Family != "kubernetes" {
		t.Fatalf("suggestion = %#v", suggestion)
	}
}

func TestSuggestedTargetResponseForPVC(t *testing.T) {
	response, ok := SuggestedTargetResponse(SuggestedTarget{
		Family:    "kubernetes-pvc",
		Name:      "web-data",
		Namespace: "prod",
		Command:   "kubectl describe pvc -n prod web-data",
	})
	if !ok {
		t.Fatal("SuggestedTargetResponse() = false, want true")
	}
	if response.IntentID != "inspect_k8s_pvc_describe" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Command != "kubectl describe pvc -n prod web-data" {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestSuggestedTargetResponseForDeployment(t *testing.T) {
	response, ok := SuggestedTargetResponse(SuggestedTarget{
		Family:    "kubernetes-deployment",
		Name:      "web",
		Namespace: "prod",
		Command:   "kubectl describe deployment -n prod web",
	})
	if !ok {
		t.Fatal("SuggestedTargetResponse() = false, want true")
	}
	if response.IntentID != "inspect_k8s_deployment_describe" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Command != "kubectl describe deployment -n prod web" {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestSuggestedTargetResponseForService(t *testing.T) {
	response, ok := SuggestedTargetResponse(SuggestedTarget{
		Family:    "kubernetes-service",
		Name:      "api",
		Namespace: "prod",
		Command:   "kubectl describe service -n prod api",
	})
	if !ok {
		t.Fatal("SuggestedTargetResponse() = false, want true")
	}
	if response.IntentID != "inspect_k8s_service_describe" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if response.Command != "kubectl describe service -n prod api" {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestApplySuggestedTargetResetsStaleContext(t *testing.T) {
	thread := StateThread{
		Query:         "why is worker 2 not up?",
		LastCommand:   "systemctl status worker2 --no-pager -l",
		Executed:      []string{"systemctl status worker2 --no-pager -l"},
		LastDiscovery: []string{"No matching systemd unit name found for worker2."},
		LastFindings:  []string{"Live service evidence: The requested unit could not be found on this host."},
		LastWarnings:  []string{"query was ambiguous"},
		History:       []ProbeRecord{{Command: "systemctl status worker2 --no-pager -l"}},
		FamilyScores:  map[string]float64{"service": -1.0, "kubernetes": 1.2},
		CauseScores:   map[string]float64{"service_unit_missing": 2.4, "service_process_failure": 0.8, "kubernetes_crashloop": 0.4},
	}

	updated := ApplySuggestedTarget(thread, SuggestedTarget{
		Family: "kubernetes",
		Name:   "worker-2",
	})

	if updated.LastCommand != "" || len(updated.Executed) != 0 {
		t.Fatalf("expected stale executed command state to be cleared: %#v", updated)
	}
	if len(updated.LastDiscovery) != 0 || len(updated.LastFindings) != 0 || len(updated.History) != 0 {
		t.Fatalf("expected stale discovery/findings/history to be cleared: %#v", updated)
	}
	if len(updated.LastWarnings) != 1 || updated.LastWarnings[0] != "operator adopted discovered target: worker-2 (kubernetes)" {
		t.Fatalf("LastWarnings = %#v", updated.LastWarnings)
	}
	if _, ok := updated.CauseScores["service_unit_missing"]; ok {
		t.Fatalf("CauseScores = %#v, expected service_unit_missing to be retired", updated.CauseScores)
	}
	if _, ok := updated.CauseScores["service_process_failure"]; ok {
		t.Fatalf("CauseScores = %#v, expected service_process_failure to be retired", updated.CauseScores)
	}
	if updated.CauseScores["kubernetes_crashloop"] != 0.4 {
		t.Fatalf("CauseScores = %#v, expected compatible kubernetes cause to remain", updated.CauseScores)
	}
}
