package metadata

import "testing"

func TestJobStatus_String(t *testing.T) {
	tests := []struct {
		status JobStatus
		want   string
	}{
		{JobStatusPendingDispatch, "pending_dispatch"},
		{JobStatusDispatched, "dispatched"},
		{JobStatusDispatchFailed, "dispatch_failed"},
		{JobStatusRunning, "running"},
		{JobStatusCompleted, "completed"},
		{JobStatusFailed, "failed"},
		{JobStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestJobStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
		want   bool
	}{
		{"pending_dispatch is valid", JobStatusPendingDispatch, true},
		{"dispatched is valid", JobStatusDispatched, true},
		{"dispatch_failed is valid", JobStatusDispatchFailed, true},
		{"running is valid", JobStatusRunning, true},
		{"completed is valid", JobStatusCompleted, true},
		{"failed is valid", JobStatusFailed, true},
		{"cancelled is valid", JobStatusCancelled, true},
		{"legacy pending invalid", JobStatus("pending"), false},
		{"invalid status", JobStatus("invalid"), false},
		{"empty status", JobStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
		want   bool
	}{
		{"pending_dispatch is not terminal", JobStatusPendingDispatch, false},
		{"dispatched is not terminal", JobStatusDispatched, false},
		{"dispatch_failed is not terminal", JobStatusDispatchFailed, false},
		{"running is not terminal", JobStatusRunning, false},
		{"completed is terminal", JobStatusCompleted, true},
		{"failed is terminal", JobStatusFailed, true},
		{"cancelled is terminal", JobStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.want {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name string
		from JobStatus
		to   JobStatus
		want bool
	}{
		{"pending_dispatch to dispatched", JobStatusPendingDispatch, JobStatusDispatched, true},
		{"pending_dispatch to dispatch_failed", JobStatusPendingDispatch, JobStatusDispatchFailed, true},
		{"pending_dispatch to cancelled", JobStatusPendingDispatch, JobStatusCancelled, true},
		{"pending_dispatch to running", JobStatusPendingDispatch, JobStatusRunning, false},
		{"dispatch_failed to pending_dispatch", JobStatusDispatchFailed, JobStatusPendingDispatch, true},
		{"dispatched to running", JobStatusDispatched, JobStatusRunning, true},
		{"dispatched to cancelled", JobStatusDispatched, JobStatusCancelled, true},
		{"dispatched to failed", JobStatusDispatched, JobStatusFailed, true},
		{"running to completed", JobStatusRunning, JobStatusCompleted, true},
		{"running to failed", JobStatusRunning, JobStatusFailed, true},
		{"failed to pending_dispatch", JobStatusFailed, JobStatusPendingDispatch, true},
		{"failed to running", JobStatusFailed, JobStatusRunning, false},
		{"completed to running", JobStatusCompleted, JobStatusRunning, false},
		{"invalid target", JobStatusPendingDispatch, JobStatus("nope"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
				t.Errorf("CanTransitionTo() = %v, want %v", got, tt.want)
			}
		})
	}
}
