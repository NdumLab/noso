package troubleshoot

import "testing"

func TestResolveClarificationUsesLatestThread(t *testing.T) {
	state := State{
		Threads: []StateThread{
			{Query: "why is worker 2 not up?"},
		},
	}
	thread, hint, ok := ResolveClarification(state, "it's actually a pod")
	if !ok {
		t.Fatal("ResolveClarification() = false, want true")
	}
	if thread.Query != "why is worker 2 not up?" {
		t.Fatalf("thread.Query = %q", thread.Query)
	}
	if hint.Family != "kubernetes" || hint.Label != "pod" {
		t.Fatalf("hint = %#v", hint)
	}
}

func TestApplyClarificationHintReweightsFamilies(t *testing.T) {
	thread := StateThread{
		Query:        "why is worker 2 not up?",
		FamilyScores: map[string]float64{"service": 1.2, "runtime": 0.5, "kubernetes": 0.1},
		CauseScores:  map[string]float64{"service_unit_missing": 2.4},
	}
	updated := ApplyClarificationHint(thread, ClarificationHint{Family: "kubernetes", Label: "pod"})
	if updated.FamilyScores["kubernetes"] <= updated.FamilyScores["service"] {
		t.Fatalf("FamilyScores = %#v", updated.FamilyScores)
	}
	if updated.CauseScores["service_unit_missing"] != 0 {
		t.Fatalf("CauseScores = %#v", updated.CauseScores)
	}
	if len(updated.LastWarnings) == 0 {
		t.Fatal("expected clarification warning to be recorded")
	}
}

func TestApplyClarificationQueryAppendsFamilyLabel(t *testing.T) {
	updated := ApplyClarificationQuery("why is worker 2 not up?", ClarificationHint{Family: "runtime", Label: "container"})
	if updated != "why is worker 2 not up? container" {
		t.Fatalf("updated query = %q", updated)
	}
}
