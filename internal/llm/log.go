package llm

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NdumLab/noso/pkg/models"
)

type RequestLogger struct {
	mu   sync.Mutex
	path string
}

type RequestLogRecord struct {
	Timestamp          string  `json:"timestamp"`
	Provider           string  `json:"provider"`
	Model              string  `json:"model"`
	Query              string  `json:"query"`
	NeedsClarification bool    `json:"needs_clarification"`
	CandidateCount     int     `json:"candidate_count"`
	TopIntent          string  `json:"top_intent,omitempty"`
	TopConfidence      float64 `json:"top_confidence,omitempty"`
	Error              string  `json:"error,omitempty"`
}

type LogFilter struct {
	Match             string
	Limit             int
	Since             time.Time
	Provider          string
	ErrorOnly         bool
	ClarificationOnly bool
}

type LogSummary struct {
	Total           int            `json:"total"`
	Clarifications  int            `json:"clarifications"`
	Errors          int            `json:"errors"`
	ProviderCounts  map[string]int `json:"provider_counts"`
	ErrorCounts     map[string]int `json:"error_counts"`
	TopIntentCounts map[string]int `json:"top_intent_counts"`
}

func NewRequestLogger(path string) (*RequestLogger, error) {
	if path == "" {
		return nil, nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &RequestLogger{path: path}, nil
}

func (l *RequestLogger) Append(provider, model, query string, resp models.LLMInterpretResponse, err error) error {
	if l == nil {
		return nil
	}
	record := RequestLogRecord{
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
		Provider:           provider,
		Model:              model,
		Query:              sanitizeField(query, 240),
		NeedsClarification: resp.NeedsClarification,
		CandidateCount:     len(resp.Candidates),
	}
	if len(resp.Candidates) > 0 {
		record.TopIntent = resp.Candidates[0].Intent
		record.TopConfidence = resp.Candidates[0].Confidence
	}
	if err != nil {
		record.Error = DescribeError(err)
	}
	data, marshalErr := json.Marshal(record)
	if marshalErr != nil {
		return marshalErr
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	f, openErr := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if openErr != nil {
		return openErr
	}
	defer f.Close()
	_, writeErr := f.Write(append(data, '\n'))
	return writeErr
}

func ReadLog(path string) ([]RequestLogRecord, error) {
	if path == "" {
		return nil, nil
	}

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var records []RequestLogRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record RequestLogRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func FilterLogs(records []RequestLogRecord, match string, limit int) []RequestLogRecord {
	return FilterLogsWith(records, LogFilter{
		Match: match,
		Limit: limit,
	})
}

func FilterLogsSince(records []RequestLogRecord, match string, limit int, since time.Time) []RequestLogRecord {
	return FilterLogsWith(records, LogFilter{
		Match: match,
		Limit: limit,
		Since: since,
	})
}

func FilterLogsWith(records []RequestLogRecord, filter LogFilter) []RequestLogRecord {
	match := strings.ToLower(strings.TrimSpace(filter.Match))
	provider := strings.ToLower(strings.TrimSpace(filter.Provider))
	var filtered []RequestLogRecord
	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		if !filter.Since.IsZero() {
			ts, err := time.Parse(time.RFC3339, record.Timestamp)
			if err != nil || ts.Before(filter.Since) {
				continue
			}
		}
		if provider != "" && strings.ToLower(record.Provider) != provider {
			continue
		}
		if filter.ErrorOnly && strings.TrimSpace(record.Error) == "" {
			continue
		}
		if filter.ClarificationOnly && !record.NeedsClarification {
			continue
		}
		if match != "" {
			blob := strings.ToLower(record.Query + "\n" + record.Provider + "\n" + record.Model + "\n" + record.TopIntent + "\n" + record.Error)
			if !strings.Contains(blob, match) {
				continue
			}
		}
		filtered = append(filtered, record)
		if filter.Limit > 0 && len(filtered) >= filter.Limit {
			break
		}
	}
	return filtered
}

func ParseSince(value string, now time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts, nil
	}
	if dur, err := time.ParseDuration(value); err == nil {
		return now.Add(-dur), nil
	}
	return time.Time{}, fmt.Errorf("invalid since value %q: use RFC3339 or a duration like 15m or 2h", value)
}

func RenderLogs(records []RequestLogRecord, asJSON bool) (string, error) {
	if asJSON {
		return RenderLogsFormat(records, "json")
	}
	return RenderLogsFormat(records, "text")
}

func RenderLogsFormat(records []RequestLogRecord, format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		return renderLogsText(records), nil
	case "json":
		data, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	case "markdown":
		return renderLogsMarkdown(records), nil
	case "csv":
		return renderLogsCSV(records)
	default:
		return "", fmt.Errorf("unsupported llm-log format %q: use text, json, markdown, or csv", format)
	}
}

func renderLogsText(records []RequestLogRecord) string {
	if len(records) == 0 {
		return "No local LLM log entries matched.\n"
	}

	var b strings.Builder
	for i, record := range records {
		clarify := "no"
		if record.NeedsClarification {
			clarify = "yes"
		}
		b.WriteString("[" + record.Timestamp + "] " + record.Query + "\n")
		b.WriteString("Provider: " + record.Provider + "  Model: " + record.Model + "\n")
		b.WriteString("Clarification: " + clarify + "  Candidates: " + itoa(record.CandidateCount) + "\n")
		if record.TopIntent != "" {
			b.WriteString("Top intent: " + record.TopIntent + "  Confidence: " + formatConfidence(record.TopConfidence) + "\n")
		}
		if record.Error != "" {
			b.WriteString("Error: " + record.Error + "\n")
		}
		if i < len(records)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func renderLogsMarkdown(records []RequestLogRecord) string {
	if len(records) == 0 {
		return "No local LLM log entries matched.\n"
	}

	var b strings.Builder
	b.WriteString("| Timestamp | Provider | Model | Query | Clarification | Candidates | Top Intent | Confidence | Error |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, record := range records {
		clarify := "no"
		if record.NeedsClarification {
			clarify = "yes"
		}
		b.WriteString("| " + escapeMarkdownCell(record.Timestamp) +
			" | " + escapeMarkdownCell(record.Provider) +
			" | " + escapeMarkdownCell(record.Model) +
			" | " + escapeMarkdownCell(record.Query) +
			" | " + clarify +
			" | " + itoa(record.CandidateCount) +
			" | " + escapeMarkdownCell(record.TopIntent) +
			" | " + escapeMarkdownCell(formatConfidence(record.TopConfidence)) +
			" | " + escapeMarkdownCell(record.Error) + " |\n")
	}
	return b.String()
}

func renderLogsCSV(records []RequestLogRecord) (string, error) {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if err := w.Write([]string{"timestamp", "provider", "model", "query", "needs_clarification", "candidate_count", "top_intent", "top_confidence", "error"}); err != nil {
		return "", err
	}
	for _, record := range records {
		if err := w.Write([]string{
			record.Timestamp,
			record.Provider,
			record.Model,
			record.Query,
			strconv.FormatBool(record.NeedsClarification),
			itoa(record.CandidateCount),
			record.TopIntent,
			formatConfidence(record.TopConfidence),
			record.Error,
		}); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return b.String(), nil
}

func RenderLogSummary(summary LogSummary, asJSON bool) (string, error) {
	if asJSON {
		return RenderLogSummaryFormat(summary, "json")
	}
	return RenderLogSummaryFormat(summary, "text")
}

func RenderLogSummaryFormat(summary LogSummary, format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		return renderLogSummaryText(summary), nil
	case "json":
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	case "markdown":
		return renderLogSummaryMarkdown(summary), nil
	case "csv":
		return renderLogSummaryCSV(summary)
	default:
		return "", fmt.Errorf("unsupported llm-log format %q: use text, json, markdown, or csv", format)
	}
}

func SummarizeLogs(records []RequestLogRecord) LogSummary {
	summary := LogSummary{
		ProviderCounts:  map[string]int{},
		ErrorCounts:     map[string]int{},
		TopIntentCounts: map[string]int{},
	}
	for _, record := range records {
		summary.Total++
		if record.NeedsClarification {
			summary.Clarifications++
		}
		if record.Provider != "" {
			summary.ProviderCounts[record.Provider]++
		}
		if record.TopIntent != "" {
			summary.TopIntentCounts[record.TopIntent]++
		}
		if record.Error != "" {
			summary.Errors++
			summary.ErrorCounts[record.Error]++
		}
	}
	return summary
}

func renderLogSummaryText(summary LogSummary) string {
	var b strings.Builder
	b.WriteString("LLM Log Summary\n")
	b.WriteString("Total: " + itoa(summary.Total) + "\n")
	b.WriteString("Clarifications: " + itoa(summary.Clarifications) + "\n")
	b.WriteString("Errors: " + itoa(summary.Errors) + "\n")
	writeCountMap(&b, "Providers", summary.ProviderCounts)
	writeCountMap(&b, "Errors by Type", summary.ErrorCounts)
	writeCountMap(&b, "Top Intents", summary.TopIntentCounts)
	return b.String()
}

func renderLogSummaryMarkdown(summary LogSummary) string {
	var b strings.Builder
	b.WriteString("## LLM Log Summary\n\n")
	b.WriteString("| Metric | Count |\n")
	b.WriteString("| --- | --- |\n")
	b.WriteString("| Total | " + itoa(summary.Total) + " |\n")
	b.WriteString("| Clarifications | " + itoa(summary.Clarifications) + " |\n")
	b.WriteString("| Errors | " + itoa(summary.Errors) + " |\n\n")
	writeMarkdownCountMap(&b, "Providers", summary.ProviderCounts)
	writeMarkdownCountMap(&b, "Errors by Type", summary.ErrorCounts)
	writeMarkdownCountMap(&b, "Top Intents", summary.TopIntentCounts)
	return b.String()
}

func renderLogSummaryCSV(summary LogSummary) (string, error) {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if err := w.Write([]string{"section", "key", "count"}); err != nil {
		return "", err
	}
	for _, row := range []struct {
		key   string
		count int
	}{
		{key: "total", count: summary.Total},
		{key: "clarifications", count: summary.Clarifications},
		{key: "errors", count: summary.Errors},
	} {
		if err := w.Write([]string{"totals", row.key, itoa(row.count)}); err != nil {
			return "", err
		}
	}
	writeCSVCountMap(w, "providers", summary.ProviderCounts)
	writeCSVCountMap(w, "errors_by_type", summary.ErrorCounts)
	writeCSVCountMap(w, "top_intents", summary.TopIntentCounts)
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return b.String(), nil
}

func writeCountMap(b *strings.Builder, title string, counts map[string]int) {
	if len(counts) == 0 {
		return
	}
	b.WriteString(title + ":\n")
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b.WriteString("  " + key + ": " + itoa(counts[key]) + "\n")
	}
}

func writeMarkdownCountMap(b *strings.Builder, title string, counts map[string]int) {
	if len(counts) == 0 {
		return
	}
	b.WriteString("### " + title + "\n\n")
	b.WriteString("| Key | Count |\n")
	b.WriteString("| --- | --- |\n")
	for _, key := range sortedKeys(counts) {
		b.WriteString("| " + escapeMarkdownCell(key) + " | " + itoa(counts[key]) + " |\n")
	}
	b.WriteString("\n")
}

func writeCSVCountMap(w *csv.Writer, section string, counts map[string]int) {
	for _, key := range sortedKeys(counts) {
		_ = w.Write([]string{section, key, itoa(counts[key])})
	}
}

func sortedKeys(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func escapeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}

func formatConfidence(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
