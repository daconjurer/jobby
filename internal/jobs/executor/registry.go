package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/daconjurer/jobby/internal/jobs/pulsar"
)

// handlerFunc is the type-erased handler signature stored in the registry.
type handlerFunc func(ctx context.Context, jobID string, payload json.RawMessage) error

// Registry maps job names to type-erased handlers with panic recovery.
type Registry struct {
	handlers map[string]handlerFunc
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]handlerFunc),
	}
}

// Register adds a typed handler for a job name.
// Internally wraps the handler with JSON decoding and panic recovery.
func Register[T any](r *Registry, name string, exec JobExecutor[T]) {
	if r.handlers == nil {
		r.handlers = make(map[string]handlerFunc)
	}

	r.handlers[name] = func(ctx context.Context, jobID string, payload json.RawMessage) error {
		// Create a temporary JobMessage for DecodePayload
		msg := pulsar.JobMessage{
			JobID:   jobID,
			Name:    name,
			Payload: payload,
		}

		// Decode the payload into the typed value
		data, err := pulsar.DecodePayload[T](msg)
		if err != nil {
			return fmt.Errorf("decode payload for job %s: %w", name, err)
		}

		// Execute the handler with panic recovery
		return exec.Execute(ctx, jobID, data)
	}
}

// Run executes a handler for the given job name with panic recovery.
// Returns an error if the handler is not found or if execution fails.
func (r *Registry) Run(ctx context.Context, name, jobID string, payload json.RawMessage) (err error) {
	handler, ok := r.handlers[name]
	if !ok {
		return fmt.Errorf("unknown job handler: %s", name)
	}

	// Panic recovery wrapper
	defer func() {
		if rec := recover(); rec != nil {
			stackTrace := debug.Stack()
			err = fmt.Errorf("panic in handler %s: %v\nStack trace:\n%s", name, rec, stackTrace)
		}
	}()

	return handler(ctx, jobID, payload)
}

// HasHandler checks if a handler is registered for the given job name.
func (r *Registry) HasHandler(name string) bool {
	_, ok := r.handlers[name]
	return ok
}
