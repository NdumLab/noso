package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/NdumLab/noso/internal/config"
	"github.com/NdumLab/noso/pkg/models"
)

type Client struct {
	endpoint string
	client   *http.Client
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		endpoint: cfg.LLMEndpoint,
		client: &http.Client{
			Timeout: time.Duration(cfg.LLMTimeoutMS) * time.Millisecond,
		},
	}
}

func BuildRequest(query string, env models.Environment) models.LLMInterpretRequest {
	tools := make([]string, 0, len(env.Commands))
	for name, info := range env.Commands {
		if info.Exists {
			tools = append(tools, name)
		}
	}
	sort.Strings(tools)

	return models.LLMInterpretRequest{
		Version: "1",
		Query:   query,
		Mode:    "assist",
		Environment: models.LLMEnvironment{
			OSFamily:       env.Distro,
			PackageManager: env.PackageManager,
			Shell:          env.Shell,
			AvailableTools: tools,
			IsRHEL9:        env.IsRHEL9,
		},
		Hints: models.LLMInterpretHints{
			MaxCandidates:      3,
			AllowClarification: true,
		},
	}
}

func (c *Client) Interpret(ctx context.Context, req models.LLMInterpretRequest) (models.LLMInterpretResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return models.LLMInterpretResponse{}, fmt.Errorf("marshal llm request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(data))
	if err != nil {
		return models.LLMInterpretResponse{}, fmt.Errorf("build llm request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return models.LLMInterpretResponse{}, classifyRequestError("client interpret", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.LLMInterpretResponse{}, classifyStatusError("client interpret", resp.StatusCode, fmt.Errorf("llm returned status %d", resp.StatusCode))
	}

	var parsed models.LLMInterpretResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return models.LLMInterpretResponse{}, classifyDecodeError("client interpret", fmt.Errorf("decode llm response: %w", err))
	}
	validated, err := ValidateResponse(parsed, req)
	if err != nil {
		return models.LLMInterpretResponse{}, classifyDecodeError("client interpret", fmt.Errorf("validate llm response: %w", err))
	}
	return validated, nil
}
