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
