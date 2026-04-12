package llm

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/NdumLab/noso/pkg/models"
)

func TestReadFilterRenderLogs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noso-llm.jsonl")
	logger, err := NewRequestLogger(path)
	if err != nil {
		t.Fatalf("NewRequestLogger() error = %v", err)
	}
	if err := logger.Append("ollama", "qwen", "worker2 status", RequestLogResponse(false, "service_status", 0.91), nil); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := os.WriteFile(path, append(mustReadFile(t, path), []byte("not-json\n")...), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	records, err := ReadLog(path)
	if err != nil {
		t.Fatalf("ReadLog() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}

	filtered := FilterLogs(records, "worker2", 10)
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}

	rendered, err := RenderLogs(filtered, false)
	if err != nil {
		t.Fatalf("RenderLogs() error = %v", err)
	}
	if !strings.Contains(rendered, "Top intent: service_status") {
		t.Fatalf("rendered = %q", rendered)
	}
}

func TestParseSince(t *testing.T) {
	now := time.Date(2026, 4, 11, 5, 0, 0, 0, time.UTC)
	since, err := ParseSince("2h", now)
	if err != nil {
		t.Fatalf("ParseSince() error = %v", err)
	}
	want := time.Date(2026, 4, 11, 3, 0, 0, 0, time.UTC)
	if !since.Equal(want) {
		t.Fatalf("since = %v, want %v", since, want)
	}
}

func TestFilterLogsSince(t *testing.T) {
	records := []RequestLogRecord{
		{Timestamp: "2026-04-11T03:00:00Z", Query: "old"},
		{Timestamp: "2026-04-11T05:00:00Z", Query: "new"},
	}
	since := time.Date(2026, 4, 11, 4, 0, 0, 0, time.UTC)
	filtered := FilterLogsSince(records, "", 10, since)
	if len(filtered) != 1 || filtered[0].Query != "new" {
		t.Fatalf("filtered = %#v", filtered)
	}
}

func TestFilterLogsWithProviderAndErrorOnly(t *testing.T) {
	records := []RequestLogRecord{
		{Timestamp: "2026-04-11T05:00:00Z", Provider: "heuristic", Query: "ok"},
		{Timestamp: "2026-04-11T05:01:00Z", Provider: "ollama", Query: "error", Error: "local llm timed out"},
		{Timestamp: "2026-04-11T05:02:00Z", Provider: "heuristic", Query: "error", Error: "local llm is unavailable"},
	}
	filtered := FilterLogsWith(records, LogFilter{
		Provider:  "ollama",
		ErrorOnly: true,
		Limit:     10,
	})
	if len(filtered) != 1 || filtered[0].Provider != "ollama" || filtered[0].Error == "" {
		t.Fatalf("filtered = %#v", filtered)
	}
}

func TestFilterLogsWithClarificationOnly(t *testing.T) {
	records := []RequestLogRecord{
		{Timestamp: "2026-04-11T05:00:00Z", Query: "plain", NeedsClarification: false},
		{Timestamp: "2026-04-11T05:01:00Z", Query: "ambiguous", NeedsClarification: true},
	}
	filtered := FilterLogsWith(records, LogFilter{
		ClarificationOnly: true,
		Limit:             10,
	})
	if len(filtered) != 1 || filtered[0].Query != "ambiguous" {
		t.Fatalf("filtered = %#v", filtered)
	}
}

func TestSummarizeAndRenderLogSummary(t *testing.T) {
	records := []RequestLogRecord{
		{Provider: "heuristic", NeedsClarification: true, TopIntent: "service_troubleshoot"},
		{Provider: "ollama", Error: "local llm timed out"},
		{Provider: "ollama", TopIntent: "service_status"},
	}
	summary := SummarizeLogs(records)
	if summary.Total != 3 || summary.Clarifications != 1 || summary.Errors != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	if !reflect.DeepEqual(summary.ProviderCounts, map[string]int{"heuristic": 1, "ollama": 2}) {
		t.Fatalf("ProviderCounts = %#v", summary.ProviderCounts)
	}
	rendered, err := RenderLogSummary(summary, false)
	if err != nil {
		t.Fatalf("RenderLogSummary() error = %v", err)
	}
	if !strings.Contains(rendered, "LLM Log Summary") || !strings.Contains(rendered, "Providers:") {
		t.Fatalf("rendered = %q", rendered)
	}
}

func TestRenderLogsFormatMarkdownAndCSV(t *testing.T) {
	records := []RequestLogRecord{{
		Timestamp:          "2026-04-11T05:01:00Z",
		Provider:           "ollama",
		Model:              "qwen2.5:7b-instruct",
		Query:              "why is worker 2 not up?",
		NeedsClarification: true,
		CandidateCount:     2,
		TopIntent:          "service_troubleshoot",
		TopConfidence:      0.68,
		Error:              "local llm timed out",
	}}

	markdown, err := RenderLogsFormat(records, "markdown")
	if err != nil {
		t.Fatalf("RenderLogsFormat(markdown) error = %v", err)
	}
	if !strings.Contains(markdown, "| Timestamp | Provider |") || !strings.Contains(markdown, "why is worker 2 not up?") {
		t.Fatalf("markdown = %q", markdown)
	}

	csvOut, err := RenderLogsFormat(records, "csv")
	if err != nil {
		t.Fatalf("RenderLogsFormat(csv) error = %v", err)
	}
	if !strings.Contains(csvOut, "timestamp,provider,model,query") || !strings.Contains(csvOut, "why is worker 2 not up?") {
		t.Fatalf("csvOut = %q", csvOut)
	}
}

func TestRenderLogSummaryFormatMarkdownAndCSV(t *testing.T) {
	summary := SummarizeLogs([]RequestLogRecord{
		{Provider: "heuristic", NeedsClarification: true, TopIntent: "service_troubleshoot"},
		{Provider: "ollama", Error: "local llm timed out"},
	})

	markdown, err := RenderLogSummaryFormat(summary, "markdown")
	if err != nil {
		t.Fatalf("RenderLogSummaryFormat(markdown) error = %v", err)
	}
	if !strings.Contains(markdown, "## LLM Log Summary") || !strings.Contains(markdown, "| Metric | Count |") {
		t.Fatalf("markdown = %q", markdown)
	}

	csvOut, err := RenderLogSummaryFormat(summary, "csv")
	if err != nil {
		t.Fatalf("RenderLogSummaryFormat(csv) error = %v", err)
	}
	if !strings.Contains(csvOut, "section,key,count") || !strings.Contains(csvOut, "providers,ollama,1") {
		t.Fatalf("csvOut = %q", csvOut)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	return data
}

func RequestLogResponse(clarify bool, intent string, confidence float64) models.LLMInterpretResponse {
	return models.LLMInterpretResponse{
		NeedsClarification: clarify,
		Candidates: []models.LLMIntentCandidate{{
			Intent:     intent,
			Confidence: confidence,
		}},
	}
}
