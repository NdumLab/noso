package incident

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NdumLab/noso/internal/troubleshoot"
	"github.com/NdumLab/noso/pkg/models"
)

type State struct {
	Incidents []Record `json:"incidents"`
}

type Record struct {
	ID           string            `json:"id"`
	Query        string            `json:"query"`
	Status       string            `json:"status"`
	Source       string            `json:"source,omitempty"`
	Severity     string            `json:"severity,omitempty"`
	Summary      string            `json:"summary,omitempty"`
	Fingerprint  string            `json:"fingerprint,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	StartedAt    string            `json:"started_at"`
	UpdatedAt    string            `json:"updated_at"`
	ResolvedAt   string            `json:"resolved_at,omitempty"`
	Resolution   string            `json:"resolution,omitempty"`
	ActiveFamily string            `json:"active_family,omitempty"`
	ActiveTarget string            `json:"active_target,omitempty"`
	Namespace    string            `json:"namespace,omitempty"`
	LastIntentID string            `json:"last_intent_id,omitempty"`
	LastCommand  string            `json:"last_command,omitempty"`
	LastSummary  string            `json:"last_summary,omitempty"`
	LikelyCauses []string          `json:"likely_causes,omitempty"`
	LastFindings []string          `json:"last_findings,omitempty"`
	LastWarnings []string          `json:"last_warnings,omitempty"`
	NextSteps    []string          `json:"next_steps,omitempty"`
	ProbeHistory []ProbeRecord     `json:"probe_history,omitempty"`
}

type Alert struct {
	Query       string            `json:"query"`
	Source      string            `json:"source,omitempty"`
	Severity    string            `json:"severity,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	Fingerprint string            `json:"fingerprint,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type ProbeRecord struct {
	Timestamp string   `json:"timestamp"`
	Command   string   `json:"command,omitempty"`
	Summary   string   `json:"summary,omitempty"`
	Findings  []string `json:"findings,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

func LoadState(path string) (State, error) {
	if strings.TrimSpace(path) == "" {
		return State{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

func SaveState(path string, state State) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func UpdateFromTroubleshoot(state State, thread troubleshoot.StateThread, response models.Response) State {
	now := time.Now().UTC().Format(time.RFC3339)
	id := normalizeKey(thread.Query)
	if id == "" {
		id = normalizeKey(thread.Query + " " + response.IntentID)
	}
	record := Record{
		ID:           id,
		Query:        thread.Query,
		Status:       "open",
		StartedAt:    now,
		UpdatedAt:    now,
		ActiveFamily: thread.ActiveFamily,
		ActiveTarget: thread.ActiveTarget,
		Namespace:    thread.ActiveNamespace,
		LastIntentID: response.IntentID,
		LastCommand:  response.Command,
		LastSummary:  summarize(response),
		LikelyCauses: append([]string{}, response.LikelyCauses...),
		LastFindings: append([]string{}, response.Findings...),
		LastWarnings: append([]string{}, response.Warnings...),
		NextSteps:    append([]string{}, response.NextSteps...),
		ProbeHistory: flattenProbeHistory(thread.History),
	}

	replaced := false
	for i, existing := range state.Incidents {
		if existing.ID != id {
			continue
		}
		record.StartedAt = existing.StartedAt
		if existing.Status == "resolved" {
			record.Status = existing.Status
			record.ResolvedAt = existing.ResolvedAt
			record.Resolution = existing.Resolution
		}
		state.Incidents[i] = record
		replaced = true
		break
	}
	if !replaced {
		state.Incidents = append([]Record{record}, state.Incidents...)
	}
	return state
}

func Resolve(state State, query string, resolution string) State {
	key := normalizeKey(query)
	now := time.Now().UTC().Format(time.RFC3339)
	for i, incident := range state.Incidents {
		if normalizeKey(incident.Query) != key {
			continue
		}
		state.Incidents[i].Status = "resolved"
		state.Incidents[i].UpdatedAt = now
		state.Incidents[i].ResolvedAt = now
		state.Incidents[i].Resolution = strings.TrimSpace(resolution)
		break
	}
	return state
}

func UpsertAlert(state State, alert Alert) State {
	now := time.Now().UTC().Format(time.RFC3339)
	query := strings.TrimSpace(alert.Query)
	if query == "" {
		query = strings.TrimSpace(alert.Summary)
	}
	id := normalizeAlertIdentity(alert, query)
	record := Record{
		ID:          id,
		Query:       query,
		Status:      "open",
		Source:      strings.TrimSpace(alert.Source),
		Severity:    normalizeSeverity(alert.Severity),
		Summary:     strings.TrimSpace(alert.Summary),
		Fingerprint: strings.TrimSpace(alert.Fingerprint),
		Labels:      cloneLabels(alert.Labels),
		StartedAt:   now,
		UpdatedAt:   now,
		LastSummary: strings.TrimSpace(alert.Summary),
	}
	replaced := false
	for i, existing := range state.Incidents {
		if existing.ID != id && normalizeKey(existing.Query) != normalizeKey(query) {
			continue
		}
		record.StartedAt = existing.StartedAt
		record.ActiveFamily = existing.ActiveFamily
		record.ActiveTarget = existing.ActiveTarget
		record.Namespace = existing.Namespace
		record.LastIntentID = existing.LastIntentID
		record.LastCommand = existing.LastCommand
		record.LikelyCauses = append([]string{}, existing.LikelyCauses...)
		record.LastFindings = append([]string{}, existing.LastFindings...)
		record.LastWarnings = append([]string{}, existing.LastWarnings...)
		record.NextSteps = append([]string{}, existing.NextSteps...)
		record.ProbeHistory = append([]ProbeRecord{}, existing.ProbeHistory...)
		record.Resolution = existing.Resolution
		record.ResolvedAt = existing.ResolvedAt
		if existing.Status == "resolved" {
			record.Status = "open"
			record.Resolution = ""
			record.ResolvedAt = ""
		}
		state.Incidents[i] = record
		replaced = true
		break
	}
	if !replaced {
		state.Incidents = append([]Record{record}, state.Incidents...)
	}
	return state
}

func Reset(state State, query string) State {
	key := normalizeKey(query)
	filtered := state.Incidents[:0]
	for _, incident := range state.Incidents {
		if key != "" && normalizeKey(incident.Query) == key {
			continue
		}
		filtered = append(filtered, incident)
	}
	state.Incidents = filtered
	return state
}

func Find(state State, query string) (Record, bool) {
	key := normalizeKey(query)
	for _, incident := range state.Incidents {
		if normalizeKey(incident.Query) == key {
			return incident, true
		}
	}
	return Record{}, false
}

func flattenProbeHistory(history []troubleshoot.ProbeRecord) []ProbeRecord {
	out := make([]ProbeRecord, 0, len(history))
	for _, item := range history {
		out = append(out, ProbeRecord{
			Timestamp: item.Timestamp,
			Command:   item.Command,
			Summary:   item.Summary,
			Findings:  append([]string{}, item.Findings...),
			Warnings:  append([]string{}, item.Warnings...),
		})
	}
	return out
}

func summarize(response models.Response) string {
	switch {
	case len(response.LikelyCauses) > 0:
		return response.LikelyCauses[0]
	case len(response.Findings) > 0:
		return response.Findings[0]
	case strings.TrimSpace(response.Explanation) != "":
		return response.Explanation
	case strings.TrimSpace(response.Command) != "":
		return response.Command
	default:
		return ""
	}
}

func normalizeKey(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func normalizeSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical", "warning", "info":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return strings.TrimSpace(value)
	}
}

func normalizeAlertIdentity(alert Alert, query string) string {
	if fp := normalizeKey(alert.Fingerprint); fp != "" {
		return fp
	}
	parts := []string{
		normalizeKey(alert.Source),
		normalizeKey(query),
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func cloneLabels(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
