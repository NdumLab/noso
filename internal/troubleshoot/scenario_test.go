package troubleshoot

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/NdumLab/noso/pkg/models"
)

type scenarioStep struct {
	name    string
	planned models.Response
	run     commandRunner
	check   func(*testing.T, models.Response, StateThread)
}

func TestScenarioRegressions(t *testing.T) {
	t.Run("service_missing_then_runtime_branch", func(t *testing.T) {
		query := "why is worker 2 not up?"
		state := State{}

		steps := []scenarioStep{
			{
				name: "initial_service_probe",
				planned: models.Response{
					IntentID:    "troubleshoot_plan",
					Command:     "systemctl status worker2 --no-pager -l",
					Explanation: "Built a ranked, read-only troubleshoot plan.",
					Risk:        "Low",
					Confidence:  "Medium",
					Discovery: []string{
						"No matching systemd unit name found for worker2.",
					},
					NextSteps: []string{
						"1. Service hypothesis (0.82): run `systemctl status worker2 --no-pager -l`.",
						"Evidence follow-up: Run `journalctl -u <service> -n 50 --no-pager` for more detail if the unit is unhealthy.",
						"3. Container hypothesis (0.62): run `podman ps -a`.",
						"4. Container hypothesis (0.54): run `podman logs --tail 100 worker2`.",
					},
				},
				run: func(_ context.Context, command string) (string, error) {
					if command != "systemctl status worker2 --no-pager -l" {
						return "", fmt.Errorf("unexpected command %q", command)
					}
					return "Unit worker2.service could not be found.", nil
				},
				check: func(t *testing.T, response models.Response, thread StateThread) {
					t.Helper()
					if !containsPrefix(response.Findings, "Live service evidence: The requested unit could not be found on this host.") {
						t.Fatalf("Findings = %#v", response.Findings)
					}
					if thread.CauseScores["service_unit_missing"] <= 0 {
						t.Fatalf("CauseScores = %#v", thread.CauseScores)
					}
					if thread.FamilyScores["runtime"] <= thread.FamilyScores["service"] {
						t.Fatalf("FamilyScores = %#v", thread.FamilyScores)
					}
					if len(response.LikelyCauses) == 0 || !strings.Contains(response.LikelyCauses[0], "systemd unit is missing") {
						t.Fatalf("LikelyCauses = %#v", response.LikelyCauses)
					}
				},
			},
			{
				name: "repeat_query_advances_to_runtime",
				planned: models.Response{
					IntentID:    "troubleshoot_plan",
					Command:     "systemctl status worker2 --no-pager -l",
					Explanation: "Built a ranked, read-only troubleshoot plan.",
					Risk:        "Low",
					Confidence:  "Medium",
					Discovery: []string{
						"No matching systemd unit name found for worker2.",
					},
					NextSteps: []string{
						"1. Service hypothesis (0.82): run `systemctl status worker2 --no-pager -l`.",
						"Evidence follow-up: Run `journalctl -u <service> -n 50 --no-pager` for more detail if the unit is unhealthy.",
						"3. Container hypothesis (0.62): run `podman ps -a`.",
						"4. Container hypothesis (0.54): run `podman logs --tail 100 worker2`.",
					},
				},
				run: func(_ context.Context, command string) (string, error) {
					switch command {
					case "podman ps -a":
						return "CONTAINER ID  IMAGE   COMMAND   CREATED   STATUS                     NAMES\nabc123 app app 1m ago Exited (1) 10 seconds ago worker2", nil
					case "podman logs --tail 100 worker2":
						return "error: failed to bind socket: permission denied", nil
					default:
						return "", fmt.Errorf("unexpected command %q", command)
					}
				},
				check: func(t *testing.T, response models.Response, thread StateThread) {
					t.Helper()
					if response.Command != "podman ps -a" {
						t.Fatalf("Command = %q", response.Command)
					}
					if !containsPrefix(response.Findings, "Runtime evidence: The runtime container list shows non-running or unhealthy containers.") {
						t.Fatalf("Findings = %#v", response.Findings)
					}
					if thread.CauseScores["runtime_container_failure"] <= 0 {
						t.Fatalf("CauseScores = %#v", thread.CauseScores)
					}
					if !containsPrefix(response.LikelyCauses, "High confidence: a container is present but exiting, restarting, or otherwise unhealthy") &&
						!containsPrefix(response.LikelyCauses, "Medium confidence: a container is present but exiting, restarting, or otherwise unhealthy") {
						t.Fatalf("LikelyCauses = %#v", response.LikelyCauses)
					}
				},
			},
		}

		for _, step := range steps {
			response, thread, updated := runScenario(query, state, step.planned, step.run)
			step.check(t, response, thread)
			state = updated
		}
	})

	t.Run("kubernetes_crashloop_dependency", func(t *testing.T) {
		query := "why is web-7c5c crashing?"
		response, thread, _ := runScenario(query, State{}, models.Response{
			IntentID:    "troubleshoot_plan",
			Command:     "kubectl describe pod -n prod web-7c5c",
			Explanation: "Built a ranked, read-only troubleshoot plan.",
			Risk:        "Low",
			Confidence:  "High",
		}, func(_ context.Context, command string) (string, error) {
			switch command {
			case "kubectl describe pod -n prod web-7c5c":
				return "Containers:\n  api:\n    Container ID: containerd://123\nState: Waiting\nReason: CrashLoopBackOff\nEvents:\nWarning BackOff Back-off restarting failed container", nil
			case "kubectl logs -n prod web-7c5c -c api --tail=100":
				return "panic: failed to connect to database: connection to server at db.internal port 5432 failed", nil
			default:
				return "", fmt.Errorf("unexpected command %q", command)
			}
		})

		if thread.ActiveFamily != "kubernetes" || thread.ActiveTarget != "web-7c5c" || thread.ActiveNamespace != "prod" {
			t.Fatalf("thread target context = %#v", thread)
		}
		if thread.ActiveContainer != "api" {
			t.Fatalf("ActiveContainer = %q, want api", thread.ActiveContainer)
		}
		if thread.CauseScores["kubernetes_crashloop"] <= 0 || thread.CauseScores["dependency_database_connectivity"] <= 0 {
			t.Fatalf("CauseScores = %#v", thread.CauseScores)
		}
		if !containsPrefix(response.NextSteps, "Evidence follow-up: Run `dig +short db.internal`") {
			t.Fatalf("NextSteps = %#v", response.NextSteps)
		}
		if !containsPrefix(response.NextSteps, "Evidence follow-up: Run `nc -vz db.internal 5432`") {
			t.Fatalf("NextSteps = %#v", response.NextSteps)
		}
		if !containsPrefix(response.LikelyCauses, "Medium confidence: the workload is failing because it cannot reach its database dependency") &&
			!containsPrefix(response.LikelyCauses, "High confidence: the workload is failing because it cannot reach its database dependency") {
			t.Fatalf("LikelyCauses = %#v", response.LikelyCauses)
		}
	})

	t.Run("kubernetes_crashloop_dependency_connection_refused", func(t *testing.T) {
		query := "worker pod alert"
		response, thread, _ := runScenario(query, State{}, models.Response{
			IntentID:    "incident_bootstrap_probe",
			Command:     "kubectl describe pod -n prod worker-2",
			Explanation: "Using the incident-seeded target as the first read-only troubleshoot probe.",
			Risk:        "Low",
			Confidence:  "High",
		}, func(_ context.Context, command string) (string, error) {
			switch command {
			case "kubectl describe pod -n prod worker-2":
				return "Containers:\n  worker:\n    Container ID: containerd://123\nState: Waiting\nReason: CrashLoopBackOff\nEvents:\nWarning BackOff Back-off restarting failed container worker", nil
			case "kubectl logs -n prod worker-2 -c worker --tail=100":
				return "dial tcp db.prod.svc.cluster.local:5432: connect: connection refused", nil
			default:
				return "", fmt.Errorf("unexpected command %q", command)
			}
		})

		if thread.ActiveFamily != "kubernetes" || thread.ActiveTarget != "worker-2" || thread.ActiveNamespace != "prod" {
			t.Fatalf("thread target context = %#v", thread)
		}
		if thread.ActiveContainer != "worker" {
			t.Fatalf("ActiveContainer = %q, want worker", thread.ActiveContainer)
		}
		if thread.CauseScores["dependency_database_connectivity"] <= 0 {
			t.Fatalf("CauseScores = %#v", thread.CauseScores)
		}
		if !containsPrefix(response.NextSteps, "Evidence follow-up: Run `dig +short db.prod.svc.cluster.local`") {
			t.Fatalf("NextSteps = %#v", response.NextSteps)
		}
		if !containsPrefix(response.NextSteps, "Evidence follow-up: Run `nc -vz db.prod.svc.cluster.local 5432`") {
			t.Fatalf("NextSteps = %#v", response.NextSteps)
		}
		var eventSteps int
		for _, step := range response.NextSteps {
			if strings.Contains(step, "kubectl get events -n prod --sort-by=.metadata.creationTimestamp") {
				eventSteps++
			}
		}
		if eventSteps != 1 {
			t.Fatalf("expected one kubectl get events step, got %d in %#v", eventSteps, response.NextSteps)
		}
		if !containsPrefix(response.LikelyCauses, "Medium confidence: the workload is failing because it cannot reach its database dependency") &&
			!containsPrefix(response.LikelyCauses, "High confidence: the workload is failing because it cannot reach its database dependency") {
			t.Fatalf("LikelyCauses = %#v", response.LikelyCauses)
		}
	})

	t.Run("kubernetes_scheduler_capacity", func(t *testing.T) {
		query := "why is web-7c5c pending?"
		response, thread, _ := runScenario(query, State{}, models.Response{
			IntentID:    "troubleshoot_plan",
			Command:     "kubectl get events -n prod --sort-by=.metadata.creationTimestamp",
			Explanation: "Built a ranked, read-only troubleshoot plan.",
			Risk:        "Low",
			Confidence:  "High",
		}, func(_ context.Context, command string) (string, error) {
			if command != "kubectl get events -n prod --sort-by=.metadata.creationTimestamp" {
				return "", fmt.Errorf("unexpected command %q", command)
			}
			return "LAST SEEN TYPE REASON OBJECT MESSAGE\n20s Warning FailedScheduling pod/web-7c5c node ip-10-0-1-12 had untolerated taint {dedicated: gpu}.", nil
		})

		if thread.CauseScores["kubernetes_scheduling_capacity"] <= 0 {
			t.Fatalf("CauseScores = %#v", thread.CauseScores)
		}
		if !containsPrefix(response.NextSteps, "Evidence follow-up: Run `kubectl describe node ip-10-0-1-12`") {
			t.Fatalf("NextSteps = %#v", response.NextSteps)
		}
		if !containsPrefix(response.LikelyCauses, "Medium confidence: the pod cannot be scheduled because of capacity or placement constraints") &&
			!containsPrefix(response.LikelyCauses, "High confidence: the pod cannot be scheduled because of capacity or placement constraints") {
			t.Fatalf("LikelyCauses = %#v", response.LikelyCauses)
		}
	})
}

func runScenario(query string, state State, planned models.Response, runner commandRunner) (models.Response, StateThread, State) {
	existing, _ := FindThread(state, query)
	response := ApplyThreadContext(planned, existing)
	response = enrichWithRunner(response, runner)
	thread := PreviewThread(existing, query, response)
	response = AttachLikelyCauses(response, thread)
	state = UpdateState(state, query, response)
	thread, _ = FindThread(state, query)
	return response, thread, state
}
