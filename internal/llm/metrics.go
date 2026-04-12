package llm

import (
	"sync/atomic"

	"github.com/NdumLab/noso/pkg/models"
)

type Metrics struct {
	totalRequests         atomic.Uint64
	successfulResponses   atomic.Uint64
	clarificationCount    atomic.Uint64
	emptyCandidateCount   atomic.Uint64
	retryCount            atomic.Uint64
	timeoutErrors         atomic.Uint64
	unavailableErrors     atomic.Uint64
	transientErrors       atomic.Uint64
	invalidResponseErrors atomic.Uint64
}

type MetricsSnapshot struct {
	Provider              string `json:"provider"`
	Model                 string `json:"model"`
	TotalRequests         uint64 `json:"total_requests"`
	SuccessfulResponses   uint64 `json:"successful_responses"`
	ClarificationCount    uint64 `json:"clarification_count"`
	EmptyCandidateCount   uint64 `json:"empty_candidate_count"`
	RetryCount            uint64 `json:"retry_count"`
	TimeoutErrors         uint64 `json:"timeout_errors"`
	UnavailableErrors     uint64 `json:"unavailable_errors"`
	TransientErrors       uint64 `json:"transient_errors"`
	InvalidResponseErrors uint64 `json:"invalid_response_errors"`
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) RecordRetry() {
	if m != nil {
		m.retryCount.Add(1)
	}
}

func (m *Metrics) RecordRequest(resp models.LLMInterpretResponse) {
	if m == nil {
		return
	}
	m.totalRequests.Add(1)
	if resp.NeedsClarification {
		m.clarificationCount.Add(1)
	}
	if len(resp.Candidates) == 0 {
		m.emptyCandidateCount.Add(1)
	} else {
		m.successfulResponses.Add(1)
	}
}

func (m *Metrics) RecordError(err error) {
	if m == nil || err == nil {
		return
	}
	var kind ErrorKind
	if llmErr, ok := err.(*Error); ok {
		kind = llmErr.Kind
	}
	switch kind {
	case ErrTimeout:
		m.timeoutErrors.Add(1)
	case ErrUnavailable:
		m.unavailableErrors.Add(1)
	case ErrTransientUpstream:
		m.transientErrors.Add(1)
	case ErrInvalidResponse:
		m.invalidResponseErrors.Add(1)
	}
}

func (m *Metrics) Snapshot(provider, model string) MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{Provider: provider, Model: model}
	}
	return MetricsSnapshot{
		Provider:              provider,
		Model:                 model,
		TotalRequests:         m.totalRequests.Load(),
		SuccessfulResponses:   m.successfulResponses.Load(),
		ClarificationCount:    m.clarificationCount.Load(),
		EmptyCandidateCount:   m.emptyCandidateCount.Load(),
		RetryCount:            m.retryCount.Load(),
		TimeoutErrors:         m.timeoutErrors.Load(),
		UnavailableErrors:     m.unavailableErrors.Load(),
		TransientErrors:       m.transientErrors.Load(),
		InvalidResponseErrors: m.invalidResponseErrors.Load(),
	}
}
