package metadata

import "testing"

func TestDispatchStatus_String(t *testing.T) {
	tests := []struct {
		status DispatchStatus
		want   string
	}{
		{DispatchStatusPending, "pending_dispatch"},
		{DispatchStatusDispatched, "dispatched"},
		{DispatchStatusFailed, "dispatch_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDispatchStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status DispatchStatus
		want   bool
	}{
		{"pending_dispatch is valid", DispatchStatusPending, true},
		{"dispatched is valid", DispatchStatusDispatched, true},
		{"dispatch_failed is valid", DispatchStatusFailed, true},
		{"invalid status", DispatchStatus("invalid"), false},
		{"empty status", DispatchStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDispatchStatus_IsTerminal(t *testing.T) {
	if DispatchStatusPending.IsTerminal() {
		t.Error("pending_dispatch should not be terminal")
	}
	if DispatchStatusDispatched.IsTerminal() {
		t.Error("dispatched should not be terminal")
	}
	if DispatchStatusFailed.IsTerminal() {
		t.Error("dispatch_failed should not be terminal")
	}
}

func TestDispatchStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name string
		from DispatchStatus
		to   DispatchStatus
		want bool
	}{
		{"pending to dispatched", DispatchStatusPending, DispatchStatusDispatched, true},
		{"pending to dispatch_failed", DispatchStatusPending, DispatchStatusFailed, true},
		{"dispatch_failed to pending", DispatchStatusFailed, DispatchStatusPending, true},
		{"dispatched to pending (invalid)", DispatchStatusDispatched, DispatchStatusPending, false},
		{"dispatched to dispatch_failed (invalid)", DispatchStatusDispatched, DispatchStatusFailed, false},
		{"dispatch_failed to dispatched (invalid)", DispatchStatusFailed, DispatchStatusDispatched, false},
		{"pending to pending (invalid)", DispatchStatusPending, DispatchStatusPending, false},
		{"invalid target", DispatchStatusPending, DispatchStatus("nope"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
				t.Errorf("CanTransitionTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutionStatus_String(t *testing.T) {
	tests := []struct {
		status ExecutionStatus
		want   string
	}{
		{ExecutionStatusNotStarted, "not_started"},
		{ExecutionStatusRunning, "running"},
		{ExecutionStatusCompleted, "completed"},
		{ExecutionStatusFailed, "failed"},
		{ExecutionStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExecutionStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status ExecutionStatus
		want   bool
	}{
		{"not_started is valid", ExecutionStatusNotStarted, true},
		{"running is valid", ExecutionStatusRunning, true},
		{"completed is valid", ExecutionStatusCompleted, true},
		{"failed is valid", ExecutionStatusFailed, true},
		{"cancelled is valid", ExecutionStatusCancelled, true},
		{"invalid status", ExecutionStatus("invalid"), false},
		{"empty status", ExecutionStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutionStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name   string
		status ExecutionStatus
		want   bool
	}{
		{"not_started is not terminal", ExecutionStatusNotStarted, false},
		{"running is not terminal", ExecutionStatusRunning, false},
		{"completed is terminal", ExecutionStatusCompleted, true},
		{"failed is terminal", ExecutionStatusFailed, true},
		{"cancelled is terminal", ExecutionStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.want {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutionStatus_IsRunning(t *testing.T) {
	if ExecutionStatusNotStarted.IsRunning() {
		t.Error("not_started should not be running")
	}
	if !ExecutionStatusRunning.IsRunning() {
		t.Error("running should be running")
	}
	if ExecutionStatusCompleted.IsRunning() {
		t.Error("completed should not be running")
	}
}

func TestExecutionStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name string
		from ExecutionStatus
		to   ExecutionStatus
		want bool
	}{
		{"not_started to running", ExecutionStatusNotStarted, ExecutionStatusRunning, true},
		{"not_started to failed", ExecutionStatusNotStarted, ExecutionStatusFailed, true},
		{"not_started to cancelled", ExecutionStatusNotStarted, ExecutionStatusCancelled, true},
		{"not_started to completed (invalid)", ExecutionStatusNotStarted, ExecutionStatusCompleted, false},
		{"running to completed", ExecutionStatusRunning, ExecutionStatusCompleted, true},
		{"running to failed", ExecutionStatusRunning, ExecutionStatusFailed, true},
		{"running to cancelled", ExecutionStatusRunning, ExecutionStatusCancelled, true},
		{"running to not_started (invalid)", ExecutionStatusRunning, ExecutionStatusNotStarted, false},
		{"completed to running (invalid)", ExecutionStatusCompleted, ExecutionStatusRunning, false},
		{"failed to not_started (invalid)", ExecutionStatusFailed, ExecutionStatusNotStarted, false},
		{"cancelled to running (invalid)", ExecutionStatusCancelled, ExecutionStatusRunning, false},
		{"invalid target", ExecutionStatusNotStarted, ExecutionStatus("nope"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
				t.Errorf("CanTransitionTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobMetadataModel_DisplayStatus(t *testing.T) {
	tests := []struct {
		name            string
		dispatchStatus  DispatchStatus
		executionStatus ExecutionStatus
		want            JobStatus
	}{
		{"pending_dispatch + not_started = pending_dispatch", DispatchStatusPending, ExecutionStatusNotStarted, JobStatusPendingDispatch},
		{"dispatched + not_started = dispatched", DispatchStatusDispatched, ExecutionStatusNotStarted, JobStatusDispatched},
		{"dispatch_failed + not_started = dispatch_failed", DispatchStatusFailed, ExecutionStatusNotStarted, JobStatusDispatchFailed},
		{"any + running = running", DispatchStatusPending, ExecutionStatusRunning, JobStatusRunning},
		{"any + completed = completed", DispatchStatusDispatched, ExecutionStatusCompleted, JobStatusCompleted},
		{"any + failed = failed", DispatchStatusDispatched, ExecutionStatusFailed, JobStatusFailed},
		{"any + cancelled = cancelled", DispatchStatusDispatched, ExecutionStatusCancelled, JobStatusCancelled},
		{"pending_dispatch + running = running (executor claimed before dispatch confirm)", DispatchStatusPending, ExecutionStatusRunning, JobStatusRunning},
		{"dispatch_failed + running = running (race condition representable)", DispatchStatusFailed, ExecutionStatusRunning, JobStatusRunning},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &JobMetadataModel{
				DispatchStatus:  tt.dispatchStatus,
				ExecutionStatus: tt.executionStatus,
			}

			if got := job.DisplayStatus(); got != tt.want {
				t.Errorf("DisplayStatus() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNewJobMetadata_Defaults(t *testing.T) {
	job := NewJobMetadata("123e4567-e89b-12d3-a456-426614174000", "test", nil)

	if job.DispatchStatus != DispatchStatusPending {
		t.Errorf("DispatchStatus = %s, want %s", job.DispatchStatus, DispatchStatusPending)
	}
	if job.ExecutionStatus != ExecutionStatusNotStarted {
		t.Errorf("ExecutionStatus = %s, want %s", job.ExecutionStatus, ExecutionStatusNotStarted)
	}
	if job.GetStatus() != JobStatusPendingDispatch {
		t.Errorf("GetStatus() = %s, want %s", job.GetStatus(), JobStatusPendingDispatch)
	}
}

func TestCompositeStatus_IsValid(t *testing.T) {
	tests := []struct {
		name string
		pair CompositeStatus
		want bool
	}{
		{"valid pending pair", CompositeStatus{DispatchStatusPending, ExecutionStatusNotStarted}, true},
		{"valid running pair", CompositeStatus{DispatchStatusDispatched, ExecutionStatusRunning}, true},
		{"invalid dispatch", CompositeStatus{DispatchStatus("nope"), ExecutionStatusNotStarted}, false},
		{"invalid execution", CompositeStatus{DispatchStatusPending, ExecutionStatus("nope")}, false},
		{"both invalid", CompositeStatus{DispatchStatus("nope"), ExecutionStatus("nope")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pair.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompositeStatus_DisplayStatus(t *testing.T) {
	tests := []struct {
		name string
		pair CompositeStatus
		want JobStatus
	}{
		{"pending pair", CompositeStatus{DispatchStatusPending, ExecutionStatusNotStarted}, JobStatusPendingDispatch},
		{"dispatched pair", CompositeStatus{DispatchStatusDispatched, ExecutionStatusNotStarted}, JobStatusDispatched},
		{"dispatch_failed pair", CompositeStatus{DispatchStatusFailed, ExecutionStatusNotStarted}, JobStatusDispatchFailed},
		{"running pair", CompositeStatus{DispatchStatusDispatched, ExecutionStatusRunning}, JobStatusRunning},
		{"completed pair", CompositeStatus{DispatchStatusDispatched, ExecutionStatusCompleted}, JobStatusCompleted},
		{"failed pair", CompositeStatus{DispatchStatusDispatched, ExecutionStatusFailed}, JobStatusFailed},
		{"cancelled pair", CompositeStatus{DispatchStatusDispatched, ExecutionStatusCancelled}, JobStatusCancelled},
		{"executor claimed before dispatch confirm", CompositeStatus{DispatchStatusPending, ExecutionStatusRunning}, JobStatusRunning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pair.DisplayStatus(); got != tt.want {
				t.Errorf("DisplayStatus() = %s, want %s", got, tt.want)
			}
		})
	}
}
