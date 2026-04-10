package runbook

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	out, err := Render(Report{
		Title:    "Runbook: test",
		Summary:  "summary",
		Queries:  []string{"show disk free space"},
		Commands: []string{"df -h"},
	}, "markdown")
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(out, "# Runbook: test") {
		t.Fatalf("out = %q", out)
	}
}
