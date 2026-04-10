package history

import (
	"strings"
	"testing"

	"github.com/NdumLab/noso/pkg/models"
)

func TestRenderText(t *testing.T) {
	out, err := Render([]models.AuditRecord{{
		Timestamp: "2026-04-10T00:00:00Z",
		Query:     "show disk free space",
		Response: models.Response{
			IntentID:   "inspect_disk_free_space",
			Command:    "df -h",
			Risk:       "Low",
			Confidence: "High",
		},
	}}, false)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(out, "show disk free space") {
		t.Fatalf("out = %q", out)
	}
}

func TestRenderJSON(t *testing.T) {
	out, err := Render([]models.AuditRecord{{
		Timestamp: "2026-04-10T00:00:00Z",
		Query:     "show disk free space",
		Response:  models.Response{IntentID: "inspect_disk_free_space"},
	}}, true)
	if err != nil {
		t.Fatalf("Render() JSON error = %v", err)
	}
	if !strings.Contains(out, `"inspect_disk_free_space"`) {
		t.Fatalf("JSON out missing intent ID: %q", out)
	}
}

func TestRenderEmpty(t *testing.T) {
	out, err := Render(nil, false)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(out, "No audit history") {
		t.Fatalf("expected empty message, got %q", out)
	}
}

func TestRenderMultipleRecords(t *testing.T) {
	records := []models.AuditRecord{
		{
			Timestamp: "2026-04-10T00:00:00Z",
			Query:     "show disk free space",
			Response:  models.Response{IntentID: "inspect_disk_free_space", Risk: "Low", Confidence: "High"},
		},
		{
			Timestamp: "2026-04-10T00:01:00Z",
			Query:     "git status",
			Response:  models.Response{IntentID: "inspect_git_status", Risk: "Low", Confidence: "High"},
		},
	}
	out, err := Render(records, false)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(out, "show disk free space") || !strings.Contains(out, "git status") {
		t.Fatalf("both records should appear in output: %q", out)
	}
}

func TestRenderRecordWithWarnings(t *testing.T) {
	out, err := Render([]models.AuditRecord{{
		Timestamp: "2026-04-10T00:00:00Z",
		Query:     "docker status",
		Response: models.Response{
			IntentID:   "inspect_runtime_status",
			Risk:       "Low",
			Confidence: "Low",
			Warnings:   []string{"docker is not currently installed on this host"},
		},
	}}, false)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(out, "Warnings:") {
		t.Fatalf("expected Warnings line in output: %q", out)
	}
}

func TestRenderRecordNoCommand(t *testing.T) {
	out, err := Render([]models.AuditRecord{{
		Timestamp: "2026-04-10T00:00:00Z",
		Query:     "explain df -h",
		Response: models.Response{
			IntentID:   "explain_command",
			Command:    "",
			Risk:       "Low",
			Confidence: "Low",
		},
	}}, false)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	// Command line should be absent when Command is empty.
	if strings.Contains(out, "Command:") {
		t.Fatalf("Command line should be suppressed when empty: %q", out)
	}
}
