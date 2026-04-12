package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/NdumLab/noso/pkg/models"
)

type OllamaProvider struct {
	endpoint string
	model    string
	client   *http.Client
	retries  int
	metrics  *Metrics
}

func NewOllamaProvider(endpoint, model string, timeout time.Duration) Provider {
	return &OllamaProvider{
		endpoint: endpoint,
		model:    model,
		client:   &http.Client{Timeout: timeout},
		retries:  2,
	}
}

func (p *OllamaProvider) SetMetrics(metrics *Metrics) {
	p.metrics = metrics
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) Model() string {
	return p.model
}

func (p *OllamaProvider) Interpret(ctx context.Context, req models.LLMInterpretRequest) (models.LLMInterpretResponse, error) {
	payload := map[string]any{
		"model":  p.model,
		"stream": false,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": ollamaSystemPrompt,
			},
			{
				"role":    "user",
				"content": buildOllamaPrompt(req),
			},
		},
		"format": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status":                 map[string]any{"type": "string"},
				"needs_clarification":    map[string]any{"type": "boolean"},
				"clarification_question": map[string]any{"type": "string"},
				"candidates": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"intent":     map[string]any{"type": "string"},
							"target":     map[string]any{"type": "string"},
							"namespace":  map[string]any{"type": "string"},
							"tool_hint":  map[string]any{"type": "string"},
							"confidence": map[string]any{"type": "number"},
							"reasoning":  map[string]any{"type": "string"},
						},
						"required": []string{"intent", "confidence"},
					},
				},
			},
			"required": []string{"status", "needs_clarification", "candidates"},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return models.LLMInterpretResponse{}, fmt.Errorf("marshal ollama request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= p.retries; attempt++ {
		parsed, err := p.interpretOnce(ctx, data, req)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
		if !IsRetryable(err) || attempt == p.retries {
			break
		}
		if p.metrics != nil {
			p.metrics.RecordRetry()
		}
		select {
		case <-ctx.Done():
			return models.LLMInterpretResponse{}, classifyRequestError("ollama retry wait", ctx.Err())
		case <-time.After(time.Duration(attempt+1) * 150 * time.Millisecond):
		}
	}
	return models.LLMInterpretResponse{}, lastErr
}

func (p *OllamaProvider) interpretOnce(ctx context.Context, data []byte, req models.LLMInterpretRequest) (models.LLMInterpretResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(data))
	if err != nil {
		return models.LLMInterpretResponse{}, fmt.Errorf("build ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return models.LLMInterpretResponse{}, classifyRequestError("ollama interpret", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return models.LLMInterpretResponse{}, classifyStatusError("ollama interpret", resp.StatusCode, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body))))
	}

	var envelope struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return models.LLMInterpretResponse{}, classifyDecodeError("ollama interpret", fmt.Errorf("decode ollama envelope: %w", err))
	}

	content := strings.TrimSpace(stripCodeFences(envelope.Message.Content))
	var parsed models.LLMInterpretResponse
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return models.LLMInterpretResponse{}, classifyDecodeError("ollama interpret", fmt.Errorf("decode ollama response json: %w", err))
	}
	if parsed.Status == "" {
		parsed.Status = "ok"
	}
	validated, err := ValidateResponse(parsed, req)
	if err != nil {
		return models.LLMInterpretResponse{}, classifyDecodeError("ollama interpret", fmt.Errorf("validate ollama response: %w", err))
	}
	return validated, nil
}

const ollamaSystemPrompt = `You classify Linux and DevOps troubleshooting queries for a safe CLI.
Return JSON only.
Do not return shell commands.
Do not invent tools that are not listed.
Prefer clarification over guessing when the query is ambiguous.
Use only intents that the caller already supports.`

func buildOllamaPrompt(req models.LLMInterpretRequest) string {
	data, _ := json.Marshal(req)
	return "Classify this query into supported intent candidates and return only JSON.\n\nInput:\n" + string(data)
}

func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
