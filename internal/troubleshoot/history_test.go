package troubleshoot

import (
	"strings"
	"testing"
)

func TestFilterHistory(t *testing.T) {
	records := []HistoryRecord{
		{Query: "why is worker 2 not up?", Command: "podman ps -a", Summary: "runtime check"},
		{Query: "why is worker 2 not up?", Command: "systemctl status worker2 --no-pager -l", Summary: "service check"},
		{Query: "why is api not up?", Command: "kubectl get pods", Summary: "kubernetes check"},
	}
	filtered := FilterHistory(records, "why is worker 2 not up?", "podman", 10)
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}
	if filtered[0].Command != "podman ps -a" {
		t.Fatalf("Command = %q", filtered[0].Command)
	}
}

func TestRenderHistoryNoEntries(t *testing.T) {
	rendered, err := RenderHistory(nil, false)
	if err != nil {
		t.Fatalf("RenderHistory() error = %v", err)
	}
	if !strings.Contains(rendered, "No troubleshoot history entries matched.") {
		t.Fatalf("rendered = %q", rendered)
	}
}
