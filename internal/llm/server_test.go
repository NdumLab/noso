package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NdumLab/noso/pkg/models"
)

func TestInterpretAmbiguousWorkerQuery(t *testing.T) {
	resp := interpret(models.LLMInterpretRequest{
		Query: "why is worker 2 not up?",
		Environment: models.LLMEnvironment{
			AvailableTools: []string{"systemctl", "kubectl", "podman"},
		},
		Hints: models.LLMInterpretHints{
			MaxCandidates:      3,
			AllowClarification: true,
		},
	})

	if !resp.NeedsClarification {
		t.Fatal("NeedsClarification = false, want true")
	}
	if len(resp.Candidates) < 2 {
		t.Fatalf("len(Candidates) = %d, want at least 2", len(resp.Candidates))
	}
}

func TestInterpretExplicitServiceQuery(t *testing.T) {
	resp := interpret(models.LLMInterpretRequest{
		Query: "logs for worker2 service",
		Environment: models.LLMEnvironment{
			AvailableTools: []string{"systemctl"},
		},
	})

	if resp.NeedsClarification {
		t.Fatal("NeedsClarification = true, want false")
	}
	if len(resp.Candidates) != 1 {
		t.Fatalf("len(Candidates) = %d, want 1", len(resp.Candidates))
	}
	if got := resp.Candidates[0].Intent; got != "service_logs" {
		t.Fatalf("Intent = %q, want service_logs", got)
	}
	if !strings.Contains(resp.Candidates[0].Target, "worker2") {
		t.Fatalf("Target = %q, want worker2", resp.Candidates[0].Target)
	}
}

type stubProvider struct{}

func (stubProvider) Name() string  { return "stub" }
func (stubProvider) Model() string { return "stub-model" }
func (stubProvider) Interpret(_ context.Context, req models.LLMInterpretRequest) (models.LLMInterpretResponse, error) {
	return models.LLMInterpretResponse{
		Status: "ok",
		Candidates: []models.LLMIntentCandidate{{
			Intent:     "service_status",
			Target:     req.Query,
			Confidence: 0.9,
		}},
	}, nil
}

func TestHandlerUsesConfiguredProviderInHealth(t *testing.T) {
	server := httptest.NewServer(NewHandlerWithProvider(stubProvider{}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	body := buf.String()
	if !strings.Contains(body, `"provider":"stub"`) {
		t.Fatalf("health body = %q", body)
	}
	if !strings.Contains(body, `"model":"stub-model"`) {
		t.Fatalf("health body = %q", body)
	}
}

func TestHandlerMetricsAndLogging(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noso-llm.log")
	logger, err := NewRequestLogger(path)
	if err != nil {
		t.Fatalf("NewRequestLogger() error = %v", err)
	}
	metrics := NewMetrics()
	server := httptest.NewServer(NewHandlerWithOptions(stubProvider{}, metrics, logger))
	defer server.Close()

	body := strings.NewReader(`{"version":"1","query":"worker2 service status","mode":"assist","environment":{"available_tools":["systemctl"]},"hints":{"max_candidates":3,"allow_clarification":true}}`)
	resp, err := http.Post(server.URL+"/v1/interpret", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	metricsResp, err := http.Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("Get(metrics) error = %v", err)
	}
	defer metricsResp.Body.Close()
	var snapshot MetricsSnapshot
	if err := json.NewDecoder(metricsResp.Body).Decode(&snapshot); err != nil {
		t.Fatalf("Decode(metrics) error = %v", err)
	}
	if snapshot.TotalRequests != 1 {
		t.Fatalf("TotalRequests = %d, want 1", snapshot.TotalRequests)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), `"query":"worker2 service status"`) {
		t.Fatalf("log data = %q", string(data))
	}
}
