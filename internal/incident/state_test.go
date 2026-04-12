package incident

import (
	"path/filepath"
	"testing"

	"github.com/NdumLab/noso/internal/troubleshoot"
	"github.com/NdumLab/noso/pkg/models"
)

func TestUpdateFromTroubleshootCreatesIncident(t *testing.T) {
	thread := troubleshoot.StateThread{
		Query:           "why is worker 2 not up?",
		ActiveFamily:    "kubernetes",
		ActiveTarget:    "worker-2",
		ActiveNamespace: "prod",
		History: []troubleshoot.ProbeRecord{{
			Timestamp: "2026-04-12T18:00:00Z",
			Command:   "kubectl describe pod -n prod worker-2",
			Summary:   "CrashLoopBackOff event detected",
		}},
	}
	response := models.Response{
		IntentID:     "troubleshoot_plan",
		Command:      "kubectl describe pod -n prod worker-2",
		LikelyCauses: []string{"High confidence: the pod is crashing repeatedly after startup"},
		Findings:     []string{"Live Kubernetes evidence: worker-2 is in CrashLoopBackOff."},
		NextSteps:    []string{"Run `kubectl logs -n prod worker-2 --previous` to inspect the last failing container exit."},
	}
	state := UpdateFromTroubleshoot(State{}, thread, response)
	if len(state.Incidents) != 1 {
		t.Fatalf("len(Incidents) = %d, want 1", len(state.Incidents))
	}
	record := state.Incidents[0]
	if record.Status != "open" {
		t.Fatalf("Status = %q, want open", record.Status)
	}
	if record.ActiveTarget != "worker-2" || record.Namespace != "prod" {
		t.Fatalf("Record = %#v", record)
	}
	if len(record.ProbeHistory) != 1 || record.ProbeHistory[0].Command != "kubectl describe pod -n prod worker-2" {
		t.Fatalf("ProbeHistory = %#v", record.ProbeHistory)
	}
}

func TestParseAlertsNativeSingle(t *testing.T) {
	alerts, err := ParseAlerts([]byte(`{"query":"api availability alert","source":"alertmanager","severity":"critical","summary":"API error rate above threshold","labels":{"service":"api"}}`))
	if err != nil {
		t.Fatalf("ParseAlerts() error = %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1", len(alerts))
	}
	if alerts[0].Labels["service"] != "api" {
		t.Fatalf("Labels = %#v", alerts[0].Labels)
	}
}

func TestParseAlertsAlertmanagerPayload(t *testing.T) {
	alerts, err := ParseAlerts([]byte(`{
  "commonLabels":{"namespace":"prod"},
  "alerts":[
    {
      "status":"firing",
      "labels":{"alertname":"APIAvailability","severity":"critical","service":"api"},
      "annotations":{"summary":"API error rate above threshold"},
      "fingerprint":"abc123"
    }
  ]
}`))
	if err != nil {
		t.Fatalf("ParseAlerts() error = %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1", len(alerts))
	}
	if alerts[0].Source != "alertmanager" {
		t.Fatalf("Source = %q", alerts[0].Source)
	}
	if alerts[0].Fingerprint != "abc123" {
		t.Fatalf("Fingerprint = %q", alerts[0].Fingerprint)
	}
	if alerts[0].Labels["namespace"] != "prod" {
		t.Fatalf("Labels = %#v", alerts[0].Labels)
	}
}

func TestResolveIncident(t *testing.T) {
	state := State{Incidents: []Record{{
		ID:     "why is worker 2 not up?",
		Query:  "why is worker 2 not up?",
		Status: "open",
	}}}
	updated := Resolve(state, "why is worker 2 not up?", "deployment image fixed")
	record, ok := Find(updated, "why is worker 2 not up?")
	if !ok {
		t.Fatal("Find() = false, want true")
	}
	if record.Status != "resolved" {
		t.Fatalf("Status = %q, want resolved", record.Status)
	}
	if record.Resolution != "deployment image fixed" {
		t.Fatalf("Resolution = %q", record.Resolution)
	}
	if record.ResolvedAt == "" {
		t.Fatal("ResolvedAt should not be empty")
	}
}

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "incident-state.json")
	state := State{Incidents: []Record{{
		ID:           "why is api unavailable?",
		Query:        "why is api unavailable?",
		Status:       "open",
		StartedAt:    "2026-04-12T18:00:00Z",
		UpdatedAt:    "2026-04-12T18:05:00Z",
		ActiveFamily: "kubernetes-service",
		ActiveTarget: "api",
		Namespace:    "prod",
	}}}
	if err := SaveState(path, state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}
	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	record, ok := Find(loaded, "why is api unavailable?")
	if !ok {
		t.Fatal("Find() = false, want true")
	}
	if record.ActiveTarget != "api" || record.ActiveFamily != "kubernetes-service" {
		t.Fatalf("Record = %#v", record)
	}
}

func TestUpsertAlertCreatesIncident(t *testing.T) {
	state := UpsertAlert(State{}, Alert{
		Query:    "api availability alert",
		Source:   "alertmanager",
		Severity: "critical",
		Summary:  "API error rate above threshold",
		Labels: map[string]string{
			"service":   "api",
			"namespace": "prod",
		},
	})
	if len(state.Incidents) != 1 {
		t.Fatalf("len(Incidents) = %d, want 1", len(state.Incidents))
	}
	record := state.Incidents[0]
	if record.Source != "alertmanager" || record.Severity != "critical" {
		t.Fatalf("Record = %#v", record)
	}
	if record.Labels["service"] != "api" {
		t.Fatalf("Labels = %#v", record.Labels)
	}
}

func TestUpsertAlertReopensResolvedIncident(t *testing.T) {
	state := State{Incidents: []Record{{
		ID:         "alertmanager api availability alert",
		Query:      "api availability alert",
		Status:     "resolved",
		StartedAt:  "2026-04-12T18:00:00Z",
		UpdatedAt:  "2026-04-12T18:05:00Z",
		ResolvedAt: "2026-04-12T18:06:00Z",
		Resolution: "previous fix",
	}}}
	updated := UpsertAlert(state, Alert{
		Query:    "api availability alert",
		Source:   "alertmanager",
		Severity: "warning",
		Summary:  "API latency above threshold",
	})
	record, ok := Find(updated, "api availability alert")
	if !ok {
		t.Fatal("Find() = false, want true")
	}
	if record.Status != "open" {
		t.Fatalf("Status = %q, want open", record.Status)
	}
	if record.Resolution != "" || record.ResolvedAt != "" {
		t.Fatalf("Record = %#v", record)
	}
}
