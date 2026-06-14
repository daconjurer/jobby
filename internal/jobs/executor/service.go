package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
)

// ExecutorService coordinates job execution with metadata transitions.
type ExecutorService struct {
	lifecycle JobLifecycle
	registry  *Registry
}

// NewExecutorService creates a new executor service.
func NewExecutorService(lifecycle JobLifecycle, registry *Registry) *ExecutorService {
	return &ExecutorService{
		lifecycle: lifecycle,
		registry:  registry,
	}
}

// ExecuteJob orchestrates the full execution flow for a single job message.
// Returns nil to ack the message, non-nil error to nack (transient infrastructure failures only).
func (s *ExecutorService) ExecuteJob(ctx context.Context, jobID, name string, payload json.RawMessage) error {
	// 1. Validate args
	if err := s.validateArgs(jobID, name, payload); err != nil {
		log.Printf("error: invalid job args for %s: %v", jobID, err)
		if failErr := s.lifecycle.FailJob(ctx, jobID, err); failErr != nil {
			log.Printf("error: failed to mark job %s as failed after validation error: %v", jobID, failErr)
		}
		return nil
	}

	// 2. Check if handler exists
	if !s.registry.HasHandler(name) {
		err := fmt.Errorf("unknown job handler: %s", name)
		log.Printf("error: %v for job %s", err, jobID)
		if failErr := s.lifecycle.FailJob(ctx, jobID, err); failErr != nil {
			log.Printf("error: failed to mark job %s as failed after unknown handler: %v", jobID, failErr)
		}
		return nil
	}

	// 3. Test decode payload to catch decode errors early
	if err := s.testDecode(name, jobID, payload); err != nil {
		log.Printf("error: payload decode failed for job %s (%s): %v", jobID, name, err)
		if failErr := s.lifecycle.FailJob(ctx, jobID, err); failErr != nil {
			log.Printf("error: failed to mark job %s as failed after decode error: %v", jobID, failErr)
		}
		return nil
	}

	// 4. Start job (atomic transition dispatched → running)
	if err := s.lifecycle.StartJob(ctx, jobID); err != nil {
		log.Printf("error: failed to start job %s: %v (nack for retry)", jobID, err)
		return fmt.Errorf("start job: %w", err)
	}

	// 5. Execute handler (with panic recovery)
	handlerErr := s.registry.Run(ctx, name, jobID, payload)

	// 6. Update metadata based on result
	if handlerErr != nil {
		log.Printf("error: handler execution failed for job %s (%s): %v", jobID, name, handlerErr)
		if failErr := s.lifecycle.FailJob(ctx, jobID, handlerErr); failErr != nil {
			log.Printf("error: failed to mark job %s as failed after handler error: %v", jobID, failErr)
		}
		return nil
	}

	// 7. Mark job as completed
	if err := s.lifecycle.CompleteJob(ctx, jobID, nil); err != nil {
		log.Printf("error: failed to mark job %s as completed: %v", jobID, err)
		return fmt.Errorf("complete job: %w", err)
	}

	log.Printf("info: job %s (%s) completed successfully", jobID, name)
	return nil
}

// validateArgs checks that required fields are non-empty.
func (s *ExecutorService) validateArgs(jobID, name string, payload json.RawMessage) error {
	if strings.TrimSpace(jobID) == "" {
		return errors.New("jobID is required")
	}
	if strings.TrimSpace(name) == "" {
		return errors.New("job name is required")
	}
	if len(payload) == 0 {
		return errors.New("payload is required")
	}
	return nil
}

// testDecode attempts to decode the payload without executing the handler.
// This catches malformed JSON early and allows us to fail-fast with ack.
func (s *ExecutorService) testDecode(name, jobID string, payload json.RawMessage) error {
	// We can't actually test decode without knowing the type, but we can at least validate it's valid JSON
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	return nil
}
