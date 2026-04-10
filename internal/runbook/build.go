package runbook

import (
	"fmt"
	"strings"

	"github.com/NdumLab/noso/pkg/models"
)

type Report struct {
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Queries     []string `json:"queries"`
	Commands    []string `json:"commands"`
	Findings    []string `json:"findings"`
	Warnings    []string `json:"warnings,omitempty"`
	NextSteps   []string `json:"next_steps,omitempty"`
	RecordsUsed int      `json:"records_used"`
}

func Build(records []models.AuditRecord) Report {
	records = meaningful(records)
	report := Report{
		Title:       "Runbook Summary",
		Summary:     "No matching audit records were found.",
		RecordsUsed: len(records),
	}
	if len(records) == 0 {
		return report
	}

	report.Title = "Runbook: " + records[0].Query
	report.Summary = fmt.Sprintf("Generated from %d audit record(s). The most recent focus was: %s", len(records), records[0].Query)

	for _, record := range records {
		report.Queries = appendUnique(report.Queries, record.Query)
		if record.Response.Command != "" {
			report.Commands = appendUnique(report.Commands, record.Response.Command)
		}
		if record.Response.Explanation != "" {
			report.Findings = appendUnique(report.Findings, record.Response.Explanation)
		}
		for _, warning := range record.Response.Warnings {
			report.Warnings = appendUnique(report.Warnings, warning)
		}
		for _, step := range record.Response.NextSteps {
			report.NextSteps = appendUnique(report.NextSteps, step)
		}
	}

	if len(report.NextSteps) == 0 {
		report.NextSteps = append(report.NextSteps, "Re-run the most relevant safe inspection command from the session to confirm current state.")
	}
	return report
}

func meaningful(records []models.AuditRecord) []models.AuditRecord {
	var filtered []models.AuditRecord
	for _, record := range records {
		if record.Response.IntentID == "unsupported_query" {
			continue
		}
		filtered = append(filtered, record)
	}
	return filtered
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
