package troubleshoot

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/pkg/models"
)

func TestSpecializeInfrastructureProbesPrefersNslookupWhenDigMissing(t *testing.T) {
	response := models.Response{
		NextSteps: []string{
			"Evidence follow-up: Run `dig +short db.internal` or `nslookup db.internal` to confirm DNS resolution for the configured database endpoint.",
		},
	}
	env := models.Environment{
		Commands: map[string]models.CommandInfo{
			"nslookup": {Exists: true, Path: "/usr/bin/nslookup"},
		},
	}
	updated := SpecializeInfrastructureProbes(response, env, StateThread{})
	if !strings.Contains(updated.NextSteps[0], "Run `nslookup db.internal`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
	if strings.Contains(updated.NextSteps[0], "dig +short") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
}

func TestSpecializeInfrastructureProbesUsesBashTCPFallbackWhenNCMissing(t *testing.T) {
	response := models.Response{
		NextSteps: []string{
			"Evidence follow-up: Run `nc -vz db.internal 5432` to verify the upstream listener is reachable on the expected database port.",
		},
	}
	env := models.Environment{Shell: "/bin/bash"}
	updated := SpecializeInfrastructureProbes(response, env, StateThread{})
	if !strings.Contains(updated.NextSteps[0], "timeout 3 bash -lc '</dev/tcp/db.internal/5432'") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
}

func TestSpecializeInfrastructureProbesKeepsNCWhenAvailable(t *testing.T) {
	response := models.Response{
		NextSteps: []string{
			"Evidence follow-up: Run `nc -vz db.internal 5432` to verify the upstream listener is reachable on the expected database port.",
		},
	}
	env := models.Environment{
		Shell: "/bin/bash",
		Commands: map[string]models.CommandInfo{
			"nc": {Exists: true, Path: "/usr/bin/nc"},
		},
	}
	updated := SpecializeInfrastructureProbes(response, env, StateThread{})
	if !strings.Contains(updated.NextSteps[0], "Run `nc -vz db.internal 5432`") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
}

func TestSpecializeInfrastructureProbesUsesKubectlExecForDNSInKubernetesThread(t *testing.T) {
	response := models.Response{
		NextSteps: []string{
			"Evidence follow-up: Run `dig +short db.internal` or `nslookup db.internal` to confirm DNS resolution for the configured database endpoint.",
		},
	}
	thread := StateThread{
		ActiveFamily:    "kubernetes",
		ActiveTarget:    "web-7c5c",
		ActiveNamespace: "prod",
	}
	updated := SpecializeInfrastructureProbes(response, models.Environment{}, thread)
	if !strings.Contains(updated.NextSteps[0], "kubectl exec -n prod web-7c5c -- nslookup db.internal") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
}

func TestSpecializeInfrastructureProbesUsesKubectlExecForSocketProbeInKubernetesThread(t *testing.T) {
	response := models.Response{
		NextSteps: []string{
			"Evidence follow-up: Run `nc -vz db.internal 5432` to verify the upstream listener is reachable on the expected database port.",
		},
	}
	thread := StateThread{
		ActiveFamily:    "kubernetes",
		ActiveTarget:    "web-7c5c",
		ActiveNamespace: "prod",
	}
	updated := SpecializeInfrastructureProbes(response, models.Environment{}, thread)
	if !strings.Contains(updated.NextSteps[0], "kubectl exec -n prod web-7c5c -- sh -lc 'nc -vz db.internal 5432 || </dev/tcp/db.internal/5432'") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
}

func TestSpecializeInfrastructureProbesUsesKubectlExecContainerWhenKnown(t *testing.T) {
	response := models.Response{
		NextSteps: []string{
			"Evidence follow-up: Run `dig +short db.internal` or `nslookup db.internal` to confirm DNS resolution for the configured database endpoint.",
		},
	}
	thread := StateThread{
		ActiveFamily:    "kubernetes",
		ActiveTarget:    "web-7c5c",
		ActiveNamespace: "prod",
		ActiveContainer: "api",
	}
	updated := SpecializeInfrastructureProbes(response, models.Environment{}, thread)
	if !strings.Contains(updated.NextSteps[0], "kubectl exec -n prod web-7c5c -c api -- nslookup db.internal") {
		t.Fatalf("NextSteps = %#v", updated.NextSteps)
	}
}
