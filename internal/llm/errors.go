package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

type ErrorKind string

const (
	ErrTimeout           ErrorKind = "timeout"
	ErrUnavailable       ErrorKind = "unavailable"
	ErrTransientUpstream ErrorKind = "transient_upstream"
	ErrInvalidResponse   ErrorKind = "invalid_response"
)

type Error struct {
	Kind ErrorKind
	Op   string
	Err  error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return string(e.Kind)
	}
	return fmt.Sprintf("%s: %v", e.Kind, e.Err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func classifyRequestError(op string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.DeadlineExceeded):
		return &Error{Kind: ErrTimeout, Op: op, Err: err}
	default:
		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				return &Error{Kind: ErrTimeout, Op: op, Err: err}
			}
			return &Error{Kind: ErrTransientUpstream, Op: op, Err: err}
		}
		return &Error{Kind: ErrUnavailable, Op: op, Err: err}
	}
}

func classifyStatusError(op string, statusCode int, err error) error {
	switch {
	case statusCode == http.StatusTooManyRequests || statusCode >= 500:
		return &Error{Kind: ErrTransientUpstream, Op: op, Err: err}
	default:
		return &Error{Kind: ErrInvalidResponse, Op: op, Err: err}
	}
}

func classifyDecodeError(op string, err error) error {
	return &Error{Kind: ErrInvalidResponse, Op: op, Err: err}
}

func IsRetryable(err error) bool {
	var llmErr *Error
	if !errors.As(err, &llmErr) {
		return false
	}
	return llmErr.Kind == ErrTimeout || llmErr.Kind == ErrTransientUpstream
}

func DescribeError(err error) string {
	var llmErr *Error
	if !errors.As(err, &llmErr) {
		if err == nil {
			return ""
		}
		return err.Error()
	}

	switch llmErr.Kind {
	case ErrTimeout:
		return "local llm timed out"
	case ErrUnavailable:
		return "local llm is unavailable"
	case ErrTransientUpstream:
		return "local llm upstream returned a transient failure"
	case ErrInvalidResponse:
		return "local llm returned an invalid response"
	default:
		return err.Error()
	}
}
