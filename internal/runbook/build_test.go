package runbook

import (
	"strings"
	"testing"

	"github.com/noso-dev/noso/pkg/models"
)

func TestBuildReport(t *testing.T) {
	report := Build([]models.AuditRecord{{
		Query: "nginx is not starting",
		Response: models.Response{
			Command:     "systemctl status nginx --no-pager -l",
			Explanation: "The service is failing to start.",
			Warnings:    []string{"config file missing"},
			NextSteps:   []string{"Run journalctl -u nginx -n 50 --no-pager"},
		},
	}})
	if !strings.Contains(report.Title, "nginx") {
		t.Fatalf("Title = %q", report.Title)
	}
	if len(report.Commands) != 1 {
		t.Fatalf("Commands = %v", report.Commands)
	}
}

func TestBuildSkipsUnsupportedEntries(t *testing.T) {
	report := Build([]models.AuditRecord{
		{
			Query: "runbook --limit 1",
			Response: models.Response{
				IntentID:    "unsupported_query",
				Explanation: "unsupported",
			},
		},
		{
			Query: "nginx is not starting",
			Response: models.Response{
				IntentID:    "troubleshoot_service_failure",
				Command:     "systemctl status nginx --no-pager -l",
				Explanation: "The service is failing to start.",
			},
		},
	})
	if !strings.Contains(report.Title, "nginx") {
		t.Fatalf("Title = %q", report.Title)
	}
}
