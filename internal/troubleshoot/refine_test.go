package troubleshoot

import "testing"

func TestResolveThreadRefinementNamespace(t *testing.T) {
	state := State{
		Threads: []StateThread{{
			Query:        "why is worker 2 not up?",
			ActiveFamily: "kubernetes",
			ActiveTarget: "worker-2",
		}},
	}
	thread, refinement, ok := ResolveThreadRefinement(state, "worker-2 in prod")
	if !ok {
		t.Fatal("ResolveThreadRefinement() = false, want true")
	}
	if thread.ActiveTarget != "worker-2" {
		t.Fatalf("thread.ActiveTarget = %q", thread.ActiveTarget)
	}
	if refinement.Namespace != "prod" {
		t.Fatalf("Namespace = %q, want prod", refinement.Namespace)
	}
}

func TestResolveThreadRefinementRuntime(t *testing.T) {
	state := State{
		Threads: []StateThread{{
			Query:        "why is worker 2 not up?",
			ActiveFamily: "runtime",
			ActiveTarget: "worker2-api",
		}},
	}
	_, refinement, ok := ResolveThreadRefinement(state, "it is podman")
	if !ok {
		t.Fatal("ResolveThreadRefinement() = false, want true")
	}
	if refinement.Runtime != "podman" {
		t.Fatalf("Runtime = %q, want podman", refinement.Runtime)
	}
}

func TestApplyThreadRefinementQueryUsesActiveKubernetesTarget(t *testing.T) {
	thread := StateThread{
		Query:        "why is worker 2 not up?",
		ActiveFamily: "kubernetes",
		ActiveTarget: "worker-2",
	}
	got := ApplyThreadRefinementQuery("check worker-2", thread, QueryRefinement{Namespace: "prod"})
	want := "pod worker-2 why is worker 2 not up? namespace prod"
	if got != want {
		t.Fatalf("ApplyThreadRefinementQuery() = %q, want %q", got, want)
	}
}

func TestApplyThreadRefinementQueryUsesRuntimeHint(t *testing.T) {
	thread := StateThread{
		Query:        "why is worker 2 not up?",
		ActiveFamily: "runtime",
		ActiveTarget: "worker2-api",
	}
	got := ApplyThreadRefinementQuery("it is podman", thread, QueryRefinement{Runtime: "podman"})
	want := "container worker2-api why is worker 2 not up? podman"
	if got != want {
		t.Fatalf("ApplyThreadRefinementQuery() = %q, want %q", got, want)
	}
}

func TestApplyThreadRefinementQueryUsesActivePVCObject(t *testing.T) {
	thread := StateThread{
		Query:           "why is web-7c5c pending?",
		ActiveFamily:    "kubernetes-pvc",
		ActiveTarget:    "web-data",
		ActiveNamespace: "prod",
	}
	got := ApplyThreadRefinementQuery("check web-data", thread, QueryRefinement{})
	want := "pvc web-data why is web-7c5c pending? namespace prod"
	if got != want {
		t.Fatalf("ApplyThreadRefinementQuery() = %q, want %q", got, want)
	}
}

func TestApplyThreadRefinementQueryUsesActiveDeploymentObject(t *testing.T) {
	thread := StateThread{
		Query:           "why is web-7c5c failing?",
		ActiveFamily:    "kubernetes-deployment",
		ActiveTarget:    "web",
		ActiveNamespace: "prod",
	}
	got := ApplyThreadRefinementQuery("check web", thread, QueryRefinement{})
	want := "deployment web why is web-7c5c failing? namespace prod"
	if got != want {
		t.Fatalf("ApplyThreadRefinementQuery() = %q, want %q", got, want)
	}
}

func TestApplyThreadRefinementQueryUsesActiveServiceObject(t *testing.T) {
	thread := StateThread{
		Query:           "why is api unavailable?",
		ActiveFamily:    "kubernetes-service",
		ActiveTarget:    "api",
		ActiveNamespace: "prod",
	}
	got := ApplyThreadRefinementQuery("check api", thread, QueryRefinement{})
	want := "service api why is api unavailable? namespace prod"
	if got != want {
		t.Fatalf("ApplyThreadRefinementQuery() = %q, want %q", got, want)
	}
}

func TestApplyThreadRefinementQueryUsesActiveNodeObject(t *testing.T) {
	thread := StateThread{
		Query:        "why is web-7c5c pending?",
		ActiveFamily: "kubernetes-node",
		ActiveTarget: "ip-10-0-1-12",
	}
	got := ApplyThreadRefinementQuery("check ip-10-0-1-12", thread, QueryRefinement{})
	want := "node ip-10-0-1-12 why is web-7c5c pending?"
	if got != want {
		t.Fatalf("ApplyThreadRefinementQuery() = %q, want %q", got, want)
	}
}
