package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NdumLab/noso/pkg/models"
)

func TestAppendCreatesFileWithCorrectPermissions(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "state", "noso", "audit.log")
	logger := NewLogger(logPath)

	resp := models.Response{IntentID: "test_intent", Command: "df -h", Risk: "Low", Confidence: "High"}
	if err := logger.Append("show disk free space", resp); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	// File must be readable only by owner.
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Stat(log file) error = %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("log file permissions = %04o, want 0600", perm)
	}

	// Directory must be accessible only by owner.
	dirInfo, err := os.Stat(filepath.Dir(logPath))
	if err != nil {
		t.Fatalf("Stat(log dir) error = %v", err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("log dir permissions = %04o, want 0700", perm)
	}

	// The written record must be readable back.
	records, err := ReadAll(logPath)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
	if records[0].Query != "show disk free space" {
		t.Errorf("Query = %q", records[0].Query)
	}
	if records[0].Response.IntentID != "test_intent" {
		t.Errorf("IntentID = %q", records[0].Response.IntentID)
	}
}

func TestAppendEmptyPathIsNoop(t *testing.T) {
	logger := NewLogger("")
	resp := models.Response{IntentID: "test"}
	if err := logger.Append("query", resp); err != nil {
		t.Errorf("Append() with empty path should be a no-op, got error: %v", err)
	}
}

func TestReadAllSkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "audit.log")

	// Mix of valid, empty, and corrupt lines.
	data := `{"timestamp":"2026-04-10T00:00:00Z","query":"good record","response":{"intent_id":"ok","command":"df -h","explanation":"x","expected_output":"x","risk":"Low","confidence":"High"}}` + "\n" +
		`{not valid json}` + "\n" +
		`{"timestamp":"2026-04-10T00:01:00Z","query":"also good","response":{"intent_id":"ok2","command":"git status","explanation":"x","expected_output":"x","risk":"Low","confidence":"Low"}}` + "\n"

	if err := os.WriteFile(logPath, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	records, err := ReadAll(logPath)
	if err != nil {
		t.Fatalf("ReadAll() error = %v (should skip malformed lines)", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2 (bad line should be skipped)", len(records))
	}
}

func TestReadAllAndFilter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	data := `{"timestamp":"2026-04-10T00:00:00Z","query":"show disk free space","response":{"intent_id":"inspect_disk_free_space","command":"df -h","explanation":"x","expected_output":"x","risk":"Low","confidence":"High"}}` + "\n" +
		`{"timestamp":"2026-04-10T00:01:00Z","query":"show git log","response":{"intent_id":"inspect_git_log","command":"git log --oneline -n 10","explanation":"x","expected_output":"x","risk":"Low","confidence":"Low"}}` + "\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	records, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}

	filtered := Filter(records, "git", 1)
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}
	if filtered[0].Query != "show git log" {
		t.Fatalf("Query = %q", filtered[0].Query)
	}
}

func TestFilterNoLimit(t *testing.T) {
	records := []models.AuditRecord{
		{Query: "one"},
		{Query: "two"},
	}
	filtered := Filter(records, "", 0)
	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(filtered))
	}
	if filtered[0].Query != "two" {
		t.Fatalf("first query = %q, want newest first", filtered[0].Query)
	}
}
