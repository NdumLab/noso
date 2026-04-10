package runbook

import (
	"encoding/json"
	"fmt"
	"strings"
)

func Render(report Report, format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "markdown", "md":
		return renderMarkdown(report), nil
	case "json":
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	default:
		return "", fmt.Errorf("unsupported runbook format: %s", format)
	}
}

func renderMarkdown(report Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", report.Title)
	fmt.Fprintf(&b, "## Summary\n\n%s\n\n", report.Summary)
	writeList(&b, "Queries", report.Queries)
	writeList(&b, "Commands Used", report.Commands)
	writeList(&b, "Findings", report.Findings)
	writeList(&b, "Warnings", report.Warnings)
	writeList(&b, "Suggested Next Steps", report.NextSteps)
	return b.String()
}

func writeList(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "## %s\n\n", title)
	for _, item := range items {
		fmt.Fprintf(b, "- %s\n", item)
	}
	b.WriteString("\n")
}
