package incident

import (
	"fmt"
	"strings"

	"github.com/NdumLab/noso/internal/troubleshoot"
)

func BootstrapThread(record Record) (troubleshoot.StateThread, bool) {
	if strings.TrimSpace(record.Query) == "" {
		return troubleshoot.StateThread{}, false
	}
	if strings.TrimSpace(record.ActiveFamily) == "" &&
		strings.TrimSpace(record.ActiveTarget) == "" &&
		strings.TrimSpace(record.LastCommand) == "" {
		return troubleshoot.StateThread{}, false
	}

	thread := troubleshoot.StateThread{
		Query:           record.Query,
		IntentID:        "incident_ingest",
		ActiveFamily:    record.ActiveFamily,
		ActiveTarget:    record.ActiveTarget,
		ActiveNamespace: record.Namespace,
		LastCommand:     record.LastCommand,
		LastFindings:    append([]string{}, record.LastFindings...),
		LastWarnings:    append([]string{}, record.LastWarnings...),
		FamilyScores:    map[string]float64{},
		CauseScores:     map[string]float64{},
		History:         flattenIncidentHistory(record.ProbeHistory),
	}
	if record.ActiveFamily != "" {
		thread.FamilyScores[record.ActiveFamily] = 1.0
	}
	if discovery := bootstrapDiscovery(record); discovery != "" {
		thread.LastDiscovery = append(thread.LastDiscovery, discovery)
	}
	for _, probe := range thread.History {
		if command := strings.TrimSpace(probe.Command); command != "" {
			thread.Executed = append(thread.Executed, command)
		}
	}
	return thread, true
}

func SyncBootstrapThread(state troubleshoot.State, record Record) troubleshoot.State {
	boot, ok := BootstrapThread(record)
	if !ok {
		return state
	}
	queryKey := normalizeKey(record.Query)
	replaced := false
	updated := make([]troubleshoot.StateThread, 0, len(state.Threads)+1)
	for _, existing := range state.Threads {
		if normalizeKey(existing.Query) == queryKey || bootstrapTargetMatch(existing, record) {
			if !replaced {
				updated = append(updated, mergeBootstrapThread(existing, boot))
				replaced = true
			}
			continue
		}
		updated = append(updated, existing)
	}
	if !replaced {
		updated = append([]troubleshoot.StateThread{boot}, updated...)
	}
	if len(updated) > 10 {
		updated = updated[:10]
	}
	state.Threads = updated
	return state
}

func flattenIncidentHistory(history []ProbeRecord) []troubleshoot.ProbeRecord {
	out := make([]troubleshoot.ProbeRecord, 0, len(history))
	for _, item := range history {
		out = append(out, troubleshoot.ProbeRecord{
			Timestamp: item.Timestamp,
			Command:   item.Command,
			Summary:   item.Summary,
			Findings:  append([]string{}, item.Findings...),
			Warnings:  append([]string{}, item.Warnings...),
		})
	}
	return out
}

func bootstrapDiscovery(record Record) string {
	if strings.TrimSpace(record.ActiveTarget) == "" || strings.TrimSpace(record.ActiveFamily) == "" {
		return ""
	}
	if strings.TrimSpace(record.Namespace) != "" {
		return fmt.Sprintf("Alert labels identified %s target %s in namespace %s.", record.ActiveFamily, record.ActiveTarget, record.Namespace)
	}
	return fmt.Sprintf("Alert labels identified %s target %s.", record.ActiveFamily, record.ActiveTarget)
}

func bootstrapTargetMatch(thread troubleshoot.StateThread, record Record) bool {
	if strings.TrimSpace(thread.ActiveFamily) == "" || strings.TrimSpace(thread.ActiveTarget) == "" {
		return false
	}
	if normalizeKey(thread.ActiveFamily) != normalizeKey(record.ActiveFamily) {
		return false
	}
	if normalizeKey(thread.ActiveTarget) != normalizeKey(record.ActiveTarget) {
		return false
	}
	if normalizeKey(thread.ActiveNamespace) != normalizeKey(record.Namespace) {
		return false
	}
	return true
}

func mergeBootstrapThread(existing, boot troubleshoot.StateThread) troubleshoot.StateThread {
	if !sameBootstrapContext(existing, boot) {
		return boot
	}
	merged := existing
	merged.Query = boot.Query
	merged.IntentID = boot.IntentID
	merged.ActiveFamily = boot.ActiveFamily
	merged.ActiveTarget = boot.ActiveTarget
	merged.ActiveNamespace = boot.ActiveNamespace
	if strings.TrimSpace(boot.LastCommand) != "" {
		merged.LastCommand = boot.LastCommand
	}
	merged.LastDiscovery = mergeUnique(existing.LastDiscovery, boot.LastDiscovery)
	merged.LastFindings = mergeUnique(existing.LastFindings, boot.LastFindings)
	merged.LastWarnings = mergeUnique(existing.LastWarnings, boot.LastWarnings)
	if merged.FamilyScores == nil {
		merged.FamilyScores = map[string]float64{}
	}
	for family, score := range boot.FamilyScores {
		if merged.FamilyScores[family] < score {
			merged.FamilyScores[family] = score
		}
	}
	return merged
}

func sameBootstrapContext(existing, boot troubleshoot.StateThread) bool {
	return normalizeKey(existing.ActiveFamily) == normalizeKey(boot.ActiveFamily) &&
		normalizeKey(existing.ActiveTarget) == normalizeKey(boot.ActiveTarget) &&
		normalizeKey(existing.ActiveNamespace) == normalizeKey(boot.ActiveNamespace)
}

func mergeUnique(existing, extra []string) []string {
	out := append([]string{}, existing...)
	for _, value := range extra {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		seen := false
		for _, item := range out {
			if item == value {
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
