package executor

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// testExecutor is a simple executor for testing
type testExecutor struct {
	executeFunc func(ctx context.Context, jobID string, data testPayload) error
}

type testPayload struct {
	Value string `json:"value"`
}

func (e *testExecutor) Execute(ctx context.Context, jobID string, data testPayload) error {
	if e.executeFunc != nil {
		return e.executeFunc(ctx, jobID, data)
	}
	return nil
}

// panicExecutor always panics
type panicExecutor struct{}

func (e *panicExecutor) Execute(ctx context.Context, jobID string, data testPayload) error {
	panic("test panic")
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	exec := &testExecutor{}

	Register(r, "test-job", exec)

	if !r.HasHandler("test-job") {
		t.Error("expected handler to be registered")
	}
	if r.HasHandler("unknown-job") {
		t.Error("expected unknown handler to not be registered")
	}
}

func TestRegistry_Run_Success(t *testing.T) {
	r := NewRegistry()
	executed := false
	exec := &testExecutor{
		executeFunc: func(ctx context.Context, jobID string, data testPayload) error {
			executed = true
			if jobID != "job-123" {
				t.Errorf("expected jobID job-123, got %s", jobID)
			}
			if data.Value != "test-value" {
				t.Errorf("expected value test-value, got %s", data.Value)
			}
			return nil
		},
	}
	Register(r, "test-job", exec)

	payload := json.RawMessage(`{"value":"test-value"}`)
	err := r.Run(context.Background(), "test-job", "job-123", payload)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !executed {
		t.Error("expected handler to be executed")
	}
}

func TestRegistry_Run_UnknownHandler(t *testing.T) {
	r := NewRegistry()
	payload := json.RawMessage(`{"value":"test"}`)

	err := r.Run(context.Background(), "unknown-job", "job-123", payload)

	if err == nil {
		t.Fatal("expected error for unknown handler")
	}
	if !strings.Contains(err.Error(), "unknown job handler") {
		t.Errorf("expected 'unknown job handler' error, got: %v", err)
	}
}

func TestRegistry_Run_DecodeError(t *testing.T) {
	r := NewRegistry()
	exec := &testExecutor{}
	Register(r, "test-job", exec)

	// Invalid JSON
	payload := json.RawMessage(`{invalid json}`)
	err := r.Run(context.Background(), "test-job", "job-123", payload)

	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "decode payload") {
		t.Errorf("expected 'decode payload' error, got: %v", err)
	}
}

func TestRegistry_Run_HandlerError(t *testing.T) {
	r := NewRegistry()
	expectedErr := errors.New("handler error")
	exec := &testExecutor{
		executeFunc: func(ctx context.Context, jobID string, data testPayload) error {
			return expectedErr
		},
	}
	Register(r, "test-job", exec)

	payload := json.RawMessage(`{"value":"test"}`)
	err := r.Run(context.Background(), "test-job", "job-123", payload)

	if err == nil {
		t.Fatal("expected handler error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRegistry_Run_PanicRecovery(t *testing.T) {
	r := NewRegistry()
	exec := &panicExecutor{}
	Register(r, "panic-job", exec)

	payload := json.RawMessage(`{"value":"test"}`)
	err := r.Run(context.Background(), "panic-job", "job-123", payload)

	if err == nil {
		t.Fatal("expected panic to be recovered as error")
	}
	if !strings.Contains(err.Error(), "panic in handler") {
		t.Errorf("expected 'panic in handler' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "test panic") {
		t.Errorf("expected panic message in error, got: %v", err)
	}
}

func TestRegistry_HasHandler(t *testing.T) {
	r := NewRegistry()
	exec := &testExecutor{}

	if r.HasHandler("test-job") {
		t.Error("expected handler to not exist yet")
	}

	Register(r, "test-job", exec)

	if !r.HasHandler("test-job") {
		t.Error("expected handler to exist")
	}
	if r.HasHandler("other-job") {
		t.Error("expected other-job to not exist")
	}
}
