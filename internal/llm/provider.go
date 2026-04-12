package llm

import (
	"context"

	"github.com/NdumLab/noso/pkg/models"
)

type Provider interface {
	Name() string
	Model() string
	Interpret(ctx context.Context, req models.LLMInterpretRequest) (models.LLMInterpretResponse, error)
}

type HeuristicProvider struct{}

func NewHeuristicProvider() Provider {
	return HeuristicProvider{}
}

func (HeuristicProvider) Name() string {
	return "heuristic"
}

func (HeuristicProvider) Model() string {
	return "heuristic-local"
}

func (HeuristicProvider) Interpret(_ context.Context, req models.LLMInterpretRequest) (models.LLMInterpretResponse, error) {
	return interpret(req), nil
}
