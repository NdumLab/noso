package llm

import (
	"context"
	"errors"
	"net"
	"testing"
)

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return true }

func TestDescribeErrorTimeout(t *testing.T) {
	err := classifyRequestError("test", context.DeadlineExceeded)
	if got := DescribeError(err); got != "local llm timed out" {
		t.Fatalf("DescribeError() = %q", got)
	}
}

func TestClassifyRequestErrorTransient(t *testing.T) {
	var netErr net.Error = timeoutNetError{}
	err := classifyRequestError("test", netErr)
	if !IsRetryable(err) {
		t.Fatal("IsRetryable() = false, want true")
	}
}

func TestDescribeErrorUnavailable(t *testing.T) {
	err := classifyRequestError("test", errors.New("connection refused"))
	if got := DescribeError(err); got != "local llm is unavailable" {
		t.Fatalf("DescribeError() = %q", got)
	}
}
