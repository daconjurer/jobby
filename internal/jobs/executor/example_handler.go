package executor

import (
	"context"
	"fmt"
	"log"
)

// ExampleHandler is a simple echo handler for testing and demonstration.
// In production, handlers would contain actual business logic.
type ExampleHandler struct{}

// ExamplePayload is the expected payload structure for the example handler.
type ExamplePayload struct {
	Message string `json:"message"`
}

// Execute implements JobExecutor[ExamplePayload].
// This is a no-op handler that just logs the message.
func (h *ExampleHandler) Execute(ctx context.Context, jobID string, data ExamplePayload) error {
	log.Printf("ExampleHandler: processing job %s with message: %s", jobID, data.Message)

	// In a real handler, you would:
	// - Perform business logic (e.g., call external APIs, process data)
	// - Return error if the job should be marked as failed
	// - Return nil on success

	if data.Message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	return nil
}
