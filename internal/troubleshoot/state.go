package troubleshoot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NdumLab/noso/pkg/models"
)

type State struct {
	UpdatedAt string        `json:"updated_at"`
	Threads   []StateThread `json:"threads"`
}

type StateThread struct {
	Query        string             `json:"query"`
	IntentID     string             `json:"intent_id"`
	LastCommand  string             `json:"last_command,omitempty"`
	Executed     []string           `json:"executed,omitempty"`
	LastFindings []string           `json:"last_findings,omitempty"`
	LastWarnings []string           `json:"last_warnings,omitempty"`
	FamilyScores map[string]float64 `json:"family_scores,omitempty"`
	History      []ProbeRecord      `json:"history,omitempty"`
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
	if len(data) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

func SaveState(path string, state State) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func UpdateState(state State, query string, response models.Response) State {
	normalized := normalizeQuery(query)
	thread := StateThread{
		Query:        query,
		IntentID:     response.IntentID,
		LastCommand:  response.Command,
		LastFindings: append([]string{}, response.Findings...),
		LastWarnings: append([]string{}, response.Warnings...),
		FamilyScores: map[string]float64{},
	}
	record := newProbeRecord(response)
	if record.Timestamp != "" {
		thread.History = []ProbeRecord{record}
	}
	if response.Command != "" {
		thread.Executed = append(thread.Executed, response.Command)
	}

	var updated []StateThread
	updated = append(updated, thread)
	for _, existing := range state.Threads {
		if normalizeQuery(existing.Query) == normalized {
			thread.Executed = appendUniqueStrings(thread.Executed, existing.Executed...)
			thread.LastFindings = appendUniqueStrings(thread.LastFindings, existing.LastFindings...)
			thread.LastWarnings = appendUniqueStrings(thread.LastWarnings, existing.LastWarnings...)
			thread.FamilyScores = mergeFamilyScores(existing.FamilyScores, thread.FamilyScores)
			thread.History = append(thread.History, existing.History...)
			continue
		}
		updated = append(updated, existing)
		if len(updated) >= 10 {
			break
		}
	}
	applyFamilyScoreDelta(&thread, response)
	updated[0] = thread
	state.Threads = updated
	return state
}

func ResetState(state State, query string) State {
	if strings.TrimSpace(query) == "" {
		state.Threads = nil
		return state
	}

	normalized := normalizeQuery(query)
	filtered := make([]StateThread, 0, len(state.Threads))
	for _, thread := range state.Threads {
		if normalizeQuery(thread.Query) == normalized {
			continue
		}
		filtered = append(filtered, thread)
	}
	state.Threads = filtered
	return state
}

func FindThread(state State, query string) (StateThread, bool) {
	normalized := normalizeQuery(query)
	for _, thread := range state.Threads {
		if normalizeQuery(thread.Query) == normalized {
			return thread, true
		}
	}
	return StateThread{}, false
}

func normalizeQuery(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func appendUniqueStrings(values []string, extra ...string) []string {
	out := append([]string{}, values...)
	for _, value := range extra {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		seen := false
		for _, existing := range out {
			if existing == value {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, value)
		}
	}
	return out
}

func mergeFamilyScores(base, overlay map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		out[key] = value
	}
	return out
}

func newProbeRecord(response models.Response) ProbeRecord {
	record := ProbeRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Command:   strings.TrimSpace(response.Command),
		Findings:  append([]string{}, response.Findings...),
		Warnings:  append([]string{}, response.Warnings...),
		Summary:   summarizeProbe(response),
	}
	if record.Command == "" && record.Summary == "" && len(record.Findings) == 0 && len(record.Warnings) == 0 {
		return ProbeRecord{}
	}
	return record
}

func summarizeProbe(response models.Response) string {
	switch {
	case len(response.Findings) > 0:
		return strings.TrimSpace(response.Findings[0])
	case len(response.Warnings) > 0:
		return strings.TrimSpace(response.Warnings[0])
	case strings.TrimSpace(response.Explanation) != "":
		return strings.TrimSpace(response.Explanation)
	case strings.TrimSpace(response.IntentID) != "":
		return "Intent: " + strings.TrimSpace(response.IntentID)
	default:
		return ""
	}
}

func applyFamilyScoreDelta(thread *StateThread, response models.Response) {
	if thread.FamilyScores == nil {
		thread.FamilyScores = map[string]float64{}
	}
	family := commandFamily(response.Command)
	if family != "" && family != "other" {
		thread.FamilyScores[family] += 0.2
	}

	text := strings.ToLower(strings.Join(append(append([]string{}, response.Findings...), response.Warnings...), "\n"))
	switch {
	case strings.Contains(text, "unit could not be found"), strings.Contains(text, "unit not found"), strings.Contains(text, "service name is wrong"), strings.Contains(text, "not managed by systemd"):
		thread.FamilyScores["service"] -= 2.0
		thread.FamilyScores["runtime"] += 1.0
		thread.FamilyScores["kubernetes"] += 0.7
	case strings.Contains(text, "unit is in a failed state"), strings.Contains(text, "exit-code"):
		thread.FamilyScores["service"] += 1.2
	case strings.Contains(text, "runtime output was recognized, but no clear container state could be classified"):
		thread.FamilyScores["runtime"] -= 0.6
	case strings.Contains(text, "runtime container list shows non-running or unhealthy containers"), strings.Contains(text, "runtime log evidence"):
		thread.FamilyScores["runtime"] += 1.2
	case strings.Contains(text, "kubernetes evidence: at least one pod is not healthy"), strings.Contains(text, "kubernetes describe evidence"), strings.Contains(text, "kubernetes log evidence"):
		thread.FamilyScores["kubernetes"] += 1.2
	}

	if strings.Contains(text, "runtime probe unavailable") || strings.Contains(text, "docker is not currently installed") || strings.Contains(text, "podman is not currently installed") {
		thread.FamilyScores["runtime"] -= 1.5
	}
	if strings.Contains(text, "kubernetes probe unavailable") || strings.Contains(text, "kubectl is not currently installed") {
		thread.FamilyScores["kubernetes"] -= 1.5
	}
}
