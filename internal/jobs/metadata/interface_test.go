package metadata

import "testing"

// TestJobStatus_String tests string representation
func TestJobStatus_String(t *testing.T) {
	tests := []struct {
		status JobStatus
		want   string
	}{
		{JobStatusPending, "pending"},
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

// TestJobStatus_IsValid tests status validation
func TestJobStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
		want   bool
	}{
		{"pending is valid", JobStatusPending, true},
		{"running is valid", JobStatusRunning, true},
		{"completed is valid", JobStatusCompleted, true},
		{"failed is valid", JobStatusFailed, true},
		{"cancelled is valid", JobStatusCancelled, true},
		{"invalid status", JobStatus("invalid"), false},
		{"empty status", JobStatus(""), false},
		{"unknown status", JobStatus("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJobStatus_IsTerminal tests terminal status check
func TestJobStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
		want   bool
	}{
		{"pending is not terminal", JobStatusPending, false},
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

// TestJobStatus_CanTransitionTo tests status transition validation
func TestJobStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name   string
		from   JobStatus
		to     JobStatus
		want   bool
	}{
		// Valid transitions from pending
		{"pending to running", JobStatusPending, JobStatusRunning, true},
		{"pending to cancelled", JobStatusPending, JobStatusCancelled, true},

		// Invalid transitions from pending
		{"pending to completed", JobStatusPending, JobStatusCompleted, false},
		{"pending to failed", JobStatusPending, JobStatusFailed, false},
		{"pending to pending", JobStatusPending, JobStatusPending, false},

		// Valid transitions from running
		{"running to completed", JobStatusRunning, JobStatusCompleted, true},
		{"running to failed", JobStatusRunning, JobStatusFailed, true},
		{"running to cancelled", JobStatusRunning, JobStatusCancelled, true},

		// Invalid transitions from running
		{"running to pending", JobStatusRunning, JobStatusPending, false},
		{"running to running", JobStatusRunning, JobStatusRunning, false},

		// Terminal states cannot transition
		{"completed to running", JobStatusCompleted, JobStatusRunning, false},
		{"completed to pending", JobStatusCompleted, JobStatusPending, false},
		{"completed to failed", JobStatusCompleted, JobStatusFailed, false},
		{"completed to cancelled", JobStatusCompleted, JobStatusCancelled, false},
		{"completed to completed", JobStatusCompleted, JobStatusCompleted, false},

		{"failed to running", JobStatusFailed, JobStatusRunning, false},
		{"failed to pending", JobStatusFailed, JobStatusPending, false},
		{"failed to completed", JobStatusFailed, JobStatusCompleted, false},
		{"failed to cancelled", JobStatusFailed, JobStatusCancelled, false},
		{"failed to failed", JobStatusFailed, JobStatusFailed, false},

		{"cancelled to running", JobStatusCancelled, JobStatusRunning, false},
		{"cancelled to pending", JobStatusCancelled, JobStatusPending, false},
		{"cancelled to completed", JobStatusCancelled, JobStatusCompleted, false},
		{"cancelled to failed", JobStatusCancelled, JobStatusFailed, false},
		{"cancelled to cancelled", JobStatusCancelled, JobStatusCancelled, false},

		// Edge cases with invalid status
		{"invalid status cannot transition", JobStatus("invalid"), JobStatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
				t.Errorf("CanTransitionTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLogLevel_IsValid tests log level validation
func TestLogLevel_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  bool
	}{
		{"debug is valid", LogLevelDebug, true},
		{"info is valid", LogLevelInfo, true},
		{"warn is valid", LogLevelWarn, true},
		{"error is valid", LogLevelError, true},
		{"fatal is valid", LogLevelFatal, true},
		{"invalid level", LogLevel("invalid"), false},
		{"empty level", LogLevel(""), false},
		{"unknown level", LogLevel("trace"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestListFilter_DefaultValues tests filter struct can be initialized
func TestListFilter_DefaultValues(t *testing.T) {
	filter := ListFilter{}

	if filter.Names != nil {
		t.Error("Names should be nil by default")
	}
	if filter.Statuses != nil {
		t.Error("Statuses should be nil by default")
	}
	if filter.Tags != nil {
		t.Error("Tags should be nil by default")
	}
	if filter.MinPriority != nil {
		t.Error("MinPriority should be nil by default")
	}
	if filter.MaxPriority != nil {
		t.Error("MaxPriority should be nil by default")
	}
	if filter.Limit != 0 {
		t.Errorf("Limit = %d, want 0", filter.Limit)
	}
	if filter.Skip != 0 {
		t.Errorf("Skip = %d, want 0", filter.Skip)
	}
	if filter.SortBy != "" {
		t.Errorf("SortBy = %s, want empty", filter.SortBy)
	}
	if filter.SortDesc {
		t.Error("SortDesc should be false by default")
	}
}

// TestListFilter_WithValues tests filter with values set
func TestListFilter_WithValues(t *testing.T) {
	minPriority := 3
	maxPriority := 8

	filter := ListFilter{
		Names:       []string{"job1", "job2"},
		Statuses:    []JobStatus{JobStatusPending, JobStatusRunning},
		Tags:        []string{"tag1"},
		MinPriority: &minPriority,
		MaxPriority: &maxPriority,
		Limit:       10,
		Skip:        5,
		SortBy:      "createdAt",
		SortDesc:    true,
	}

	if len(filter.Names) != 2 {
		t.Errorf("Names length = %d, want 2", len(filter.Names))
	}
	if len(filter.Statuses) != 2 {
		t.Errorf("Statuses length = %d, want 2", len(filter.Statuses))
	}
	if *filter.MinPriority != 3 {
		t.Errorf("MinPriority = %d, want 3", *filter.MinPriority)
	}
	if *filter.MaxPriority != 8 {
		t.Errorf("MaxPriority = %d, want 8", *filter.MaxPriority)
	}
	if filter.Limit != 10 {
		t.Errorf("Limit = %d, want 10", filter.Limit)
	}
	if filter.Skip != 5 {
		t.Errorf("Skip = %d, want 5", filter.Skip)
	}
	if filter.SortBy != "createdAt" {
		t.Errorf("SortBy = %s, want createdAt", filter.SortBy)
	}
	if !filter.SortDesc {
		t.Error("SortDesc should be true")
	}
}

// TestLogFilter_DefaultValues tests log filter defaults
func TestLogFilter_DefaultValues(t *testing.T) {
	filter := LogFilter{}

	if filter.Levels != nil {
		t.Error("Levels should be nil by default")
	}
	if filter.Since != nil {
		t.Error("Since should be nil by default")
	}
	if filter.Until != nil {
		t.Error("Until should be nil by default")
	}
	if filter.Limit != 0 {
		t.Errorf("Limit = %d, want 0", filter.Limit)
	}
	if filter.Skip != 0 {
		t.Errorf("Skip = %d, want 0", filter.Skip)
	}
}

// TestLogFilter_WithValues tests log filter with values set
func TestLogFilter_WithValues(t *testing.T) {
	filter := LogFilter{
		Levels: []LogLevel{LogLevelError, LogLevelFatal},
		Limit:  50,
		Skip:   10,
	}

	if len(filter.Levels) != 2 {
		t.Errorf("Levels length = %d, want 2", len(filter.Levels))
	}
	if filter.Limit != 50 {
		t.Errorf("Limit = %d, want 50", filter.Limit)
	}
	if filter.Skip != 10 {
		t.Errorf("Skip = %d, want 10", filter.Skip)
	}
}
