package incident

import (
	"context"
	"strings"
	"testing"

	"github.com/NdumLab/noso/pkg/models"
)

func TestObserveAllowed(t *testing.T) {
	if !ObserveAllowed("journalctl -u worker2 -n 50 --no-pager") {
		t.Fatal("expected journalctl follow-up to be allowed")
	}
	if ObserveAllowed("systemctl restart worker2") {
		t.Fatal("expected mutating command to be denied")
	}
	if ObserveAllowed("kubectl exec -n prod web -- sh") {
		t.Fatal("expected in-cluster exec command to be denied")
	}
}

func TestNextObserveCommandPrefersUnreadApprovedProbe(t *testing.T) {
	record := Record{
		LastCommand: "systemctl status worker2 --no-pager -l",
		NextSteps: []string{
			"Evidence follow-up: Run `journalctl -u worker2 -n 50 --no-pager` to inspect the most recent service logs.",
			"Cause follow-up: Run `systemctl restart worker2` after the root cause is confirmed.",
		},
		ProbeHistory: []ProbeRecord{{
			Command: "systemctl status worker2 --no-pager -l",
		}},
	}
	command, err := NextObserveCommand(record)
	if err != nil {
		t.Fatalf("NextObserveCommand() error = %v", err)
	}
	if command != "journalctl -u worker2 -n 50 --no-pager" {
		t.Fatalf("command = %q", command)
	}
}

func TestNextObserveCommandFallsBackToSeededLastCommand(t *testing.T) {
	record := Record{
		Query:        "worker pod alert",
		ActiveFamily: "kubernetes",
		ActiveTarget: "worker-2",
		Namespace:    "prod",
		LastCommand:  "kubectl describe pod -n prod worker-2",
	}
	command, err := NextObserveCommand(record)
	if err != nil {
		t.Fatalf("NextObserveCommand() error = %v", err)
	}
	if command != "kubectl describe pod -n prod worker-2" {
		t.Fatalf("command = %q", command)
	}
}

func TestObserveNextWithRunner(t *testing.T) {
	record := Record{
		Query:       "why is worker 2 not up?",
		LastCommand: "systemctl status worker2 --no-pager -l",
		NextSteps: []string{
			"Evidence follow-up: Run `journalctl -u worker2 -n 50 --no-pager` to inspect the most recent service logs.",
		},
		ProbeHistory: []ProbeRecord{{
			Command: "systemctl status worker2 --no-pager -l",
		}},
	}
	response, command, err := observeNextWithRunner(record, func(_ context.Context, cmd string) (string, error) {
		if cmd != "journalctl -u worker2 -n 50 --no-pager" {
			t.Fatalf("runner cmd = %q", cmd)
		}
		return "permission denied while opening /etc/worker2/config", nil
	})
	if err != nil {
		t.Fatalf("observeNextWithRunner() error = %v", err)
	}
	if command != "journalctl -u worker2 -n 50 --no-pager" {
		t.Fatalf("command = %q", command)
	}
	if response.Command != command {
		t.Fatalf("response.Command = %q", response.Command)
	}
	if !strings.Contains(strings.ToLower(response.Explanation), "permission") {
		t.Fatalf("Explanation = %q", response.Explanation)
	}
}

func TestObserveNextWithRunnerDedupesRepeatedSteps(t *testing.T) {
	record := Record{
		Query:       "worker pod alert",
		LastCommand: "kubectl get events -n prod --sort-by=.metadata.creationTimestamp",
		NextSteps: []string{
			"Evidence follow-up: Run `kubectl describe pod -n prod worker-2` to inspect pod conditions and related events in detail.",
			"Evidence follow-up: Run `kubectl logs -n prod worker-2 -c worker --previous` to inspect the last failing container exit.",
		},
	}
	response, command, err := observeNextWithRunner(record, func(_ context.Context, cmd string) (string, error) {
		if cmd != "kubectl get events -n prod --sort-by=.metadata.creationTimestamp" {
			t.Fatalf("runner cmd = %q", cmd)
		}
		return "LAST SEEN TYPE REASON OBJECT MESSAGE\n10s Warning BackOff pod/worker-2 Back-off restarting failed container worker in pod worker-2_prod(1234)", nil
	})
	if err != nil {
		t.Fatalf("observeNextWithRunner() error = %v", err)
	}
	if command != response.Command {
		t.Fatalf("command = %q response.Command = %q", command, response.Command)
	}
	seen := map[string]int{}
	for _, step := range response.NextSteps {
		seen[step]++
	}
	for step, count := range seen {
		if count != 1 {
			t.Fatalf("step %q repeated %d times in %#v", step, count, response.NextSteps)
		}
	}
}

func TestObserveManyWithRunner(t *testing.T) {
	record := Record{
		Query:       "why is worker 2 not up?",
		LastCommand: "systemctl status worker2 --no-pager -l",
		NextSteps: []string{
			"Evidence follow-up: Run `journalctl -u worker2 -n 50 --no-pager` to inspect the most recent service logs.",
			"Evidence follow-up: Run `ss -ltnp` to inspect local listeners.",
		},
		ProbeHistory: []ProbeRecord{{
			Command: "systemctl status worker2 --no-pager -l",
		}},
	}
	var seen []string
	responses, commands, err := observeManyWithRunner(record, 2, func(_ context.Context, cmd string) (string, error) {
		seen = append(seen, cmd)
		switch cmd {
		case "journalctl -u worker2 -n 50 --no-pager", "journalctl -u <service> -n 100 --no-pager":
			return "permission denied while opening /etc/worker2/config", nil
		case "ss -ltnp":
			return "State Recv-Q Send-Q Local Address:Port Peer Address:Port Process", nil
		default:
			t.Fatalf("unexpected cmd %q", cmd)
			return "", nil
		}
	})
	if err != nil {
		t.Fatalf("observeManyWithRunner() error = %v", err)
	}
	if len(commands) != 2 || len(responses) != 2 {
		t.Fatalf("commands=%d responses=%d", len(commands), len(responses))
	}
	if !strings.HasPrefix(seen[0], "journalctl -u ") || seen[1] != "ss -ltnp" {
		t.Fatalf("seen = %#v", seen)
	}
}

func TestNextObserveCommandSkipsUnsafeStepsAndLastCommand(t *testing.T) {
	record := Record{
		LastCommand: "systemctl restart worker2",
		NextSteps: []string{
			"Cause follow-up: Run `systemctl restart worker2` after the root cause is confirmed.",
			"Evidence follow-up: Run `journalctl -u worker2 -n 50 --no-pager` to inspect the most recent service logs.",
		},
		ProbeHistory: []ProbeRecord{{
			Command: "systemctl status worker2 --no-pager -l",
		}},
	}
	command, err := NextObserveCommand(record)
	if err != nil {
		t.Fatalf("NextObserveCommand() error = %v", err)
	}
	if command != "journalctl -u worker2 -n 50 --no-pager" {
		t.Fatalf("command = %q", command)
	}
}

func TestNextObserveCommandRejectsUnsafeFallback(t *testing.T) {
	record := Record{
		Query:       "why is worker 2 not up?",
		LastCommand: "systemctl restart worker2",
		NextSteps: []string{
			"Cause follow-up: Run `systemctl restart worker2` after the root cause is confirmed.",
		},
	}
	if _, err := NextObserveCommand(record); err == nil {
		t.Fatal("expected no approved observe command when only unsafe probes remain")
	}
}

func TestApplyObservedResponsePreservesExistingSummaryForWeakResponse(t *testing.T) {
	record := Record{
		Query:       "worker pod alert",
		Summary:     "worker pod is crash looping after startup",
		LastSummary: "worker pod is crash looping after startup",
		NextSteps: []string{
			"Evidence follow-up: Run `kubectl get events --sort-by=.metadata.creationTimestamp -n prod` to inspect recent pod events.",
		},
	}
	response := applyObservedResponse(record, models.Response{
		IntentID:    "incident_bootstrap_probe",
		Command:     "kubectl describe pod -n prod worker-2",
		Explanation: "Using the incident-seeded target as the first read-only troubleshoot probe.",
	})
	if response.LastSummary != "worker pod is crash looping after startup" {
		t.Fatalf("LastSummary = %q", response.LastSummary)
	}
	if len(response.ProbeHistory) != 1 || response.ProbeHistory[0].Summary != "Using the incident-seeded target as the first read-only troubleshoot probe." {
		t.Fatalf("ProbeHistory = %#v", response.ProbeHistory)
	}
}
