package http

import (
	"testing"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

func TestFailJobMessage(t *testing.T) {
	t.Run("extracts message from errors array", func(t *testing.T) {
		msg, err := failJobMessage(FailJobRequest{
			Errors: []metadata.JobError{{Error: "boom"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if msg != "boom" {
			t.Fatalf("message=%q want boom", msg)
		}
	})

	t.Run("rejects empty error text", func(t *testing.T) {
		_, err := failJobMessage(FailJobRequest{
			Errors: []metadata.JobError{{Error: "   "}},
		})
		if err == nil {
			t.Fatal("expected error for empty errors[0].error")
		}
	})
}
