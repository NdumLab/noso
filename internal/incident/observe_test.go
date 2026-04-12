package incident

import (
	"context"
	"strings"
	"testing"
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
