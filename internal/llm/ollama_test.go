package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NdumLab/noso/pkg/models"
)

func TestOllamaProviderInterpret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"message":{"content":"{\"status\":\"ok\",\"needs_clarification\":false,\"candidates\":[{\"intent\":\"service_status\",\"target\":\"worker2\",\"confidence\":0.91}]}"}}`)
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "test-model", time.Second)
	resp, err := provider.Interpret(context.Background(), models.LLMInterpretRequest{
		Query: "worker2 service status",
	})
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	if provider.Name() != "ollama" {
		t.Fatalf("Name() = %q", provider.Name())
	}
	if got := resp.Candidates[0].Intent; got != "service_status" {
		t.Fatalf("Intent = %q, want service_status", got)
	}
}

func TestStripCodeFences(t *testing.T) {
	got := stripCodeFences("```json\n{\"status\":\"ok\"}\n```")
	if strings.TrimSpace(got) != `{"status":"ok"}` {
		t.Fatalf("stripCodeFences() = %q", got)
	}
}

func TestOllamaProviderRetriesTransientFailure(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "busy", http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, `{"message":{"content":"{\"status\":\"ok\",\"needs_clarification\":false,\"candidates\":[{\"intent\":\"service_status\",\"target\":\"worker2\",\"confidence\":0.91}]}"}}`)
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "test-model", time.Second).(*OllamaProvider)
	provider.retries = 1
	resp, err := provider.Interpret(context.Background(), models.LLMInterpretRequest{Query: "worker2 service status"})
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
	if resp.Candidates[0].Intent != "service_status" {
		t.Fatalf("Intent = %q", resp.Candidates[0].Intent)
	}
}
