package troubleshoot

import (
	"encoding/json"
	"fmt"
	"strings"
)

type HistoryRecord struct {
	Query     string   `json:"query"`
	IntentID  string   `json:"intent_id,omitempty"`
	Timestamp string   `json:"timestamp"`
	Command   string   `json:"command,omitempty"`
	Summary   string   `json:"summary,omitempty"`
	Findings  []string `json:"findings,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

func FlattenHistory(state State) []HistoryRecord {
	var records []HistoryRecord
	for _, thread := range state.Threads {
		for _, record := range thread.History {
			records = append(records, HistoryRecord{
				Query:     thread.Query,
				IntentID:  thread.IntentID,
				Timestamp: record.Timestamp,
				Command:   record.Command,
				Summary:   record.Summary,
				Findings:  append([]string{}, record.Findings...),
				Warnings:  append([]string{}, record.Warnings...),
			})
		}
	}
	return records
}

func FilterHistory(records []HistoryRecord, query string, match string, limit int) []HistoryRecord {
	query = normalizeQuery(query)
	match = strings.ToLower(strings.TrimSpace(match))
	filtered := make([]HistoryRecord, 0, len(records))
	for _, record := range records {
		if query != "" && normalizeQuery(record.Query) != query {
			continue
		}
		if match != "" && !historyMatches(record, match) {
			continue
		}
		filtered = append(filtered, record)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered
}

func RenderHistory(records []HistoryRecord, asJSON bool) (string, error) {
	if asJSON {
		data, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}
	if len(records) == 0 {
		return "No troubleshoot history entries matched.\n", nil
	}

	var b strings.Builder
	for i, record := range records {
		fmt.Fprintf(&b, "[%s] %s\n", record.Timestamp, record.Query)
		if record.IntentID != "" {
			fmt.Fprintf(&b, "Intent: %s\n", record.IntentID)
		}
		if record.Command != "" {
			fmt.Fprintf(&b, "Command: %s\n", record.Command)
		}
		if summary := preferredHistorySummary(record); summary != "" {
			fmt.Fprintf(&b, "Summary: %s\n", summary)
		}
		if len(record.Findings) > 0 {
			fmt.Fprintf(&b, "Findings: %s\n", strings.Join(record.Findings, "; "))
		}
		if len(record.Warnings) > 0 {
			fmt.Fprintf(&b, "Warnings: %s\n", strings.Join(record.Warnings, "; "))
		}
		if i < len(records)-1 {
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}

func preferredHistorySummary(record HistoryRecord) string {
	if summary := preferredCurrentSummaryEntry(record.Findings, "Previous finding:"); summary != "" {
		return summary
	}
	if summary := preferredCurrentSummaryEntry(record.Warnings, "previous thread warning:"); summary != "" {
		return summary
	}
	return strings.TrimSpace(record.Summary)
}

func historyMatches(record HistoryRecord, needle string) bool {
	haystack := strings.ToLower(strings.Join(append([]string{
		record.Query,
		record.IntentID,
		record.Command,
		record.Summary,
	}, append(record.Findings, record.Warnings...)...), "\n"))
	return strings.Contains(haystack, needle)
}
