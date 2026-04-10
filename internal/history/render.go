package history

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/noso-dev/noso/pkg/models"
)

func Render(records []models.AuditRecord, asJSON bool) (string, error) {
	if asJSON {
		data, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}

	if len(records) == 0 {
		return "No audit history entries matched.\n", nil
	}

	var b strings.Builder
	for i, record := range records {
		fmt.Fprintf(&b, "[%s] %s\n", record.Timestamp, record.Query)
		fmt.Fprintf(&b, "Intent: %s\n", record.Response.IntentID)
		if record.Response.Command != "" {
			fmt.Fprintf(&b, "Command: %s\n", record.Response.Command)
		}
		fmt.Fprintf(&b, "Risk: %s  Confidence: %s\n", record.Response.Risk, record.Response.Confidence)
		if len(record.Response.Warnings) > 0 {
			fmt.Fprintf(&b, "Warnings: %s\n", strings.Join(record.Response.Warnings, "; "))
		}
		if i < len(records)-1 {
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}
