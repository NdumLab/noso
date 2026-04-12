package incident

import (
	"encoding/json"
	"fmt"
	"strings"
)

func RenderStatus(record Record, asJSON bool) (string, error) {
	if asJSON {
		data, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}
	if record.ID == "" {
		return "No incident matched.\n", nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Incident: %s\n", record.Query)
	fmt.Fprintf(&b, "Status: %s\n", record.Status)
	fmt.Fprintf(&b, "Started: %s\n", record.StartedAt)
	fmt.Fprintf(&b, "Updated: %s\n", record.UpdatedAt)
	if record.ResolvedAt != "" {
		fmt.Fprintf(&b, "Resolved: %s\n", record.ResolvedAt)
	}
	if record.ActiveTarget != "" {
		fmt.Fprintf(&b, "Target: %s", record.ActiveTarget)
		if record.ActiveFamily != "" {
			fmt.Fprintf(&b, " (%s", record.ActiveFamily)
			if record.Namespace != "" {
				fmt.Fprintf(&b, ", namespace %s", record.Namespace)
			}
			fmt.Fprint(&b, ")")
		}
		fmt.Fprint(&b, "\n")
	}
	if record.LastSummary != "" {
		fmt.Fprintf(&b, "Summary: %s\n", record.LastSummary)
	}
	if record.LastCommand != "" {
		fmt.Fprintf(&b, "Last command: %s\n", record.LastCommand)
	}
	if len(record.LikelyCauses) > 0 {
		fmt.Fprint(&b, "Likely causes:\n")
		for _, cause := range record.LikelyCauses {
			fmt.Fprintf(&b, "- %s\n", cause)
		}
	}
	if len(record.NextSteps) > 0 {
		fmt.Fprint(&b, "Next steps:\n")
		for _, step := range record.NextSteps {
			fmt.Fprintf(&b, "- %s\n", step)
		}
	}
	if record.Resolution != "" {
		fmt.Fprintf(&b, "Resolution: %s\n", record.Resolution)
	}
	return b.String(), nil
}

func RenderHistory(records []Record, asJSON bool) (string, error) {
	if asJSON {
		data, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}
	if len(records) == 0 {
		return "No incidents matched.\n", nil
	}
	var b strings.Builder
	for i, record := range records {
		fmt.Fprintf(&b, "[%s] %s (%s)\n", record.UpdatedAt, record.Query, record.Status)
		if record.ActiveTarget != "" {
			fmt.Fprintf(&b, "Target: %s\n", record.ActiveTarget)
		}
		if record.LastSummary != "" {
			fmt.Fprintf(&b, "Summary: %s\n", record.LastSummary)
		}
		if i < len(records)-1 {
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}

func Filter(records []Record, query string, match string, status string, limit int) []Record {
	query = normalizeKey(query)
	match = strings.ToLower(strings.TrimSpace(match))
	status = strings.ToLower(strings.TrimSpace(status))
	filtered := make([]Record, 0, len(records))
	for _, record := range records {
		if query != "" && normalizeKey(record.Query) != query {
			continue
		}
		if status != "" && strings.ToLower(record.Status) != status {
			continue
		}
		if match != "" && !matches(record, match) {
			continue
		}
		filtered = append(filtered, record)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered
}

func matches(record Record, needle string) bool {
	haystack := strings.ToLower(strings.Join(append([]string{
		record.Query,
		record.Status,
		record.ActiveFamily,
		record.ActiveTarget,
		record.Namespace,
		record.LastIntentID,
		record.LastCommand,
		record.LastSummary,
		record.Resolution,
	}, append(append(record.LikelyCauses, record.LastFindings...), record.LastWarnings...)...), "\n"))
	return strings.Contains(haystack, needle)
}
