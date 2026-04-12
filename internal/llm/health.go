package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/NdumLab/noso/internal/config"
)

type HealthStatus struct {
	Enabled  bool   `json:"enabled"`
	Healthy  bool   `json:"healthy"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	Message  string `json:"message"`
}

func CheckHealth(cfg config.Config) (HealthStatus, error) {
	if !cfg.LLMEnabled {
		return HealthStatus{
			Enabled: false,
			Healthy: false,
			Message: "Local LLM fallback is disabled.",
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.LLMTimeoutMS)*time.Millisecond)
	defer cancel()

	endpoint := cfg.LLMEndpoint
	if len(endpoint) >= len("/v1/interpret") && endpoint[len(endpoint)-len("/v1/interpret"):] == "/v1/interpret" {
		endpoint = endpoint[:len(endpoint)-len("/v1/interpret")] + "/health"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return HealthStatus{Enabled: true}, classifyRequestError("llm health", err)
	}
	client := &http.Client{Timeout: time.Duration(cfg.LLMTimeoutMS) * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return HealthStatus{Enabled: true}, classifyRequestError("llm health", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return HealthStatus{Enabled: true}, classifyStatusError("llm health", resp.StatusCode, fmt.Errorf("health endpoint returned status %d", resp.StatusCode))
	}

	var parsed struct {
		Status   string `json:"status"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return HealthStatus{Enabled: true}, classifyDecodeError("llm health", fmt.Errorf("decode health response: %w", err))
	}
	if parsed.Status != "ok" {
		return HealthStatus{Enabled: true}, classifyDecodeError("llm health", fmt.Errorf("unexpected health status %q", parsed.Status))
	}

	return HealthStatus{
		Enabled:  true,
		Healthy:  true,
		Provider: parsed.Provider,
		Model:    parsed.Model,
		Message:  fmt.Sprintf("Local LLM fallback is healthy (%s/%s).", parsed.Provider, parsed.Model),
	}, nil
}
