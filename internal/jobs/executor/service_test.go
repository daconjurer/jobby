package executor

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// mockJobLifecycle is a mock implementation of JobLifecycle for testing
type mockJobLifecycle struct {
	startJobFunc    func(ctx context.Context, jobID string) error
	completeJobFunc func(ctx context.Context, jobID string, result map[string]any) error
	failJobFunc     func(ctx context.Context, jobID string, err error) error

	startJobCalls    int
	completeJobCalls int
	failJobCalls     int
}

func (m *mockJobLifecycle) StartJob(ctx context.Context, jobID string) error {
	m.startJobCalls++
	if m.startJobFunc != nil {
		return m.startJobFunc(ctx, jobID)
	}
	return nil
}

func (m *mockJobLifecycle) CompleteJob(ctx context.Context, jobID string, result map[string]any) error {
	m.completeJobCalls++
	if m.completeJobFunc != nil {
		return m.completeJobFunc(ctx, jobID, result)
	}
	return nil
}

func (m *mockJobLifecycle) FailJob(ctx context.Context, jobID string, err error) error {
	m.failJobCalls++
	if m.failJobFunc != nil {
		return m.failJobFunc(ctx, jobID, err)
	}
	return nil
}

func TestExecutorService_ExecuteJob_Success(t *testing.T) {
	mock := &mockJobLifecycle{}
	registry := NewRegistry()
	executed := false
	exec := &testExecutor{
		executeFunc: func(ctx context.Context, jobID string, data testPayload) error {
			executed = true
			return nil
		},
	}
	Register(registry, "test-job", exec)

	service := NewExecutorService(mock, registry)
	payload := json.RawMessage(`{"value":"test"}`)

	err := service.ExecuteJob(context.Background(), "job-123", "test-job", payload)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !executed {
		t.Error("expected handler to be executed")
	}
	if mock.startJobCalls != 1 {
		t.Errorf("expected StartJob to be called once, got %d", mock.startJobCalls)
	}
	if mock.completeJobCalls != 1 {
		t.Errorf("expected CompleteJob to be called once, got %d", mock.completeJobCalls)
	}
	if mock.failJobCalls != 0 {
		t.Errorf("expected FailJob to not be called, got %d", mock.failJobCalls)
	}
}

func TestExecutorService_ExecuteJob_HandlerError(t *testing.T) {
	mock := &mockJobLifecycle{}
	registry := NewRegistry()
	handlerErr := errors.New("handler error")
	exec := &testExecutor{
		executeFunc: func(ctx context.Context, jobID string, data testPayload) error {
			return handlerErr
		},
	}
	Register(registry, "test-job", exec)

	service := NewExecutorService(mock, registry)
	payload := json.RawMessage(`{"value":"test"}`)

	err := service.ExecuteJob(context.Background(), "job-123", "test-job", payload)

	if err != nil {
		t.Fatalf("expected nil (ack), got %v", err)
	}
	if mock.startJobCalls != 1 {
		t.Errorf("expected StartJob to be called once, got %d", mock.startJobCalls)
	}
	if mock.failJobCalls != 1 {
		t.Errorf("expected FailJob to be called once, got %d", mock.failJobCalls)
	}
	if mock.completeJobCalls != 0 {
		t.Errorf("expected CompleteJob to not be called, got %d", mock.completeJobCalls)
	}
}

func TestExecutorService_ExecuteJob_HandlerPanic(t *testing.T) {
	mock := &mockJobLifecycle{}
	registry := NewRegistry()
	exec := &panicExecutor{}
	Register(registry, "panic-job", exec)

	service := NewExecutorService(mock, registry)
	payload := json.RawMessage(`{"value":"test"}`)

	err := service.ExecuteJob(context.Background(), "job-123", "panic-job", payload)

	if err != nil {
		t.Fatalf("expected nil (ack), got %v", err)
	}
	if mock.startJobCalls != 1 {
		t.Errorf("expected StartJob to be called once, got %d", mock.startJobCalls)
	}
	if mock.failJobCalls != 1 {
		t.Errorf("expected FailJob to be called once (panic recovered), got %d", mock.failJobCalls)
	}
	if mock.completeJobCalls != 0 {
		t.Errorf("expected CompleteJob to not be called, got %d", mock.completeJobCalls)
	}
}

func TestExecutorService_ExecuteJob_UnknownHandler(t *testing.T) {
	mock := &mockJobLifecycle{}
	registry := NewRegistry()
	service := NewExecutorService(mock, registry)
	payload := json.RawMessage(`{"value":"test"}`)

	err := service.ExecuteJob(context.Background(), "job-123", "unknown-job", payload)

	if err != nil {
		t.Fatalf("expected nil (ack), got %v", err)
	}
	if mock.startJobCalls != 0 {
		t.Errorf("expected StartJob to not be called, got %d", mock.startJobCalls)
	}
	if mock.failJobCalls != 1 {
		t.Errorf("expected FailJob to be called once, got %d", mock.failJobCalls)
	}
}

func TestExecutorService_ExecuteJob_InvalidJobID(t *testing.T) {
	mock := &mockJobLifecycle{}
	registry := NewRegistry()
	service := NewExecutorService(mock, registry)
	payload := json.RawMessage(`{"value":"test"}`)

	err := service.ExecuteJob(context.Background(), "", "test-job", payload)

	if err != nil {
		t.Fatalf("expected nil (ack), got %v", err)
	}
	if mock.failJobCalls != 1 {
		t.Errorf("expected FailJob to be called once, got %d", mock.failJobCalls)
	}
}

func TestExecutorService_ExecuteJob_EmptyName(t *testing.T) {
	mock := &mockJobLifecycle{}
	registry := NewRegistry()
	service := NewExecutorService(mock, registry)
	payload := json.RawMessage(`{"value":"test"}`)

	err := service.ExecuteJob(context.Background(), "job-123", "", payload)

	if err != nil {
		t.Fatalf("expected nil (ack), got %v", err)
	}
	if mock.failJobCalls != 1 {
		t.Errorf("expected FailJob to be called once, got %d", mock.failJobCalls)
	}
}

func TestExecutorService_ExecuteJob_EmptyPayload(t *testing.T) {
	mock := &mockJobLifecycle{}
	registry := NewRegistry()
	service := NewExecutorService(mock, registry)

	err := service.ExecuteJob(context.Background(), "job-123", "test-job", nil)

	if err != nil {
		t.Fatalf("expected nil (ack), got %v", err)
	}
	if mock.failJobCalls != 1 {
		t.Errorf("expected FailJob to be called once, got %d", mock.failJobCalls)
	}
}

func TestExecutorService_ExecuteJob_DecodeError(t *testing.T) {
	mock := &mockJobLifecycle{}
	registry := NewRegistry()
	exec := &testExecutor{}
	Register(registry, "test-job", exec)
	service := NewExecutorService(mock, registry)

	// Invalid JSON
	payload := json.RawMessage(`{invalid}`)

	err := service.ExecuteJob(context.Background(), "job-123", "test-job", payload)

	if err != nil {
		t.Fatalf("expected nil (ack), got %v", err)
	}
	if mock.startJobCalls != 0 {
		t.Errorf("expected StartJob to not be called, got %d", mock.startJobCalls)
	}
	if mock.failJobCalls != 1 {
		t.Errorf("expected FailJob to be called once, got %d", mock.failJobCalls)
	}
}

func TestExecutorService_ExecuteJob_StartJobError(t *testing.T) {
	startErr := errors.New("mongo unavailable")
	mock := &mockJobLifecycle{
		startJobFunc: func(ctx context.Context, jobID string) error {
			return startErr
		},
	}
	registry := NewRegistry()
	exec := &testExecutor{}
	Register(registry, "test-job", exec)
	service := NewExecutorService(mock, registry)
	payload := json.RawMessage(`{"value":"test"}`)

	err := service.ExecuteJob(context.Background(), "job-123", "test-job", payload)

	// Should return error for nack
	if err == nil {
		t.Fatal("expected error (nack), got nil")
	}
	if !strings.Contains(err.Error(), "start job") {
		t.Errorf("expected 'start job' error, got: %v", err)
	}
	if mock.startJobCalls != 1 {
		t.Errorf("expected StartJob to be called once, got %d", mock.startJobCalls)
	}
	if mock.failJobCalls != 0 {
		t.Errorf("expected FailJob to not be called (infra error), got %d", mock.failJobCalls)
	}
}

func TestExecutorService_ExecuteJob_CompleteJobError(t *testing.T) {
	completeErr := errors.New("mongo unavailable")
	mock := &mockJobLifecycle{
		completeJobFunc: func(ctx context.Context, jobID string, result map[string]any) error {
			return completeErr
		},
	}
	registry := NewRegistry()
	exec := &testExecutor{}
	Register(registry, "test-job", exec)
	service := NewExecutorService(mock, registry)
	payload := json.RawMessage(`{"value":"test"}`)

	err := service.ExecuteJob(context.Background(), "job-123", "test-job", payload)

	// Should return error for nack
	if err == nil {
		t.Fatal("expected error (nack), got nil")
	}
	if !strings.Contains(err.Error(), "complete job") {
		t.Errorf("expected 'complete job' error, got: %v", err)
	}
	if mock.startJobCalls != 1 {
		t.Errorf("expected StartJob to be called once, got %d", mock.startJobCalls)
	}
	if mock.completeJobCalls != 1 {
		t.Errorf("expected CompleteJob to be called once, got %d", mock.completeJobCalls)
	}
}
