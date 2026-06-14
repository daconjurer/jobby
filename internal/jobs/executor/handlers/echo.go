package handlers

import (
	"context"
	"fmt"
	"log"
)

// EchoHandler is a simple handler that logs the payload for testing.
type EchoHandler struct{}

// EchoPayload is the typed payload for the echo handler.
type EchoPayload struct {
	Message string `json:"message"`
}

// Execute implements executor.JobExecutor for EchoPayload.
func (h *EchoHandler) Execute(ctx context.Context, jobID string, data EchoPayload) error {
	log.Printf("info: [echo handler] job %s received message: %s", jobID, data.Message)

	if data.Message == "" {
		return fmt.Errorf("message is required")
	}

	log.Printf("info: [echo handler] job %s completed successfully", jobID)
	return nil
}
