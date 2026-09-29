package mongodb

import (
	"reflect"
	"testing"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestBuildListQuery_StatusTranslation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		statuses []metadata.JobStatus
		want     bson.A
	}{
		{
			name:     "pending_dispatch",
			statuses: []metadata.JobStatus{metadata.JobStatusPendingDispatch},
			want: bson.A{
				bson.M{"dispatchStatus": metadata.DispatchStatusPending, "executionStatus": metadata.ExecutionStatusNotStarted},
			},
		},
		{
			name:     "dispatched",
			statuses: []metadata.JobStatus{metadata.JobStatusDispatched},
			want: bson.A{
				bson.M{"dispatchStatus": metadata.DispatchStatusDispatched, "executionStatus": metadata.ExecutionStatusNotStarted},
			},
		},
		{
			name:     "dispatch_failed",
			statuses: []metadata.JobStatus{metadata.JobStatusDispatchFailed},
			want: bson.A{
				bson.M{"dispatchStatus": metadata.DispatchStatusFailed, "executionStatus": metadata.ExecutionStatusNotStarted},
			},
		},
		{
			name:     "running",
			statuses: []metadata.JobStatus{metadata.JobStatusRunning},
			want: bson.A{
				bson.M{"dispatchStatus": metadata.DispatchStatusDispatched, "executionStatus": metadata.ExecutionStatusRunning},
			},
		},
		{
			name:     "completed",
			statuses: []metadata.JobStatus{metadata.JobStatusCompleted},
			want: bson.A{
				bson.M{"dispatchStatus": metadata.DispatchStatusDispatched, "executionStatus": metadata.ExecutionStatusCompleted},
			},
		},
		{
			name:     "failed",
			statuses: []metadata.JobStatus{metadata.JobStatusFailed},
			want: bson.A{
				bson.M{"dispatchStatus": metadata.DispatchStatusDispatched, "executionStatus": metadata.ExecutionStatusFailed},
			},
		},
		{
			name:     "cancelled",
			statuses: []metadata.JobStatus{metadata.JobStatusCancelled},
			want: bson.A{
				bson.M{"dispatchStatus": metadata.DispatchStatusDispatched, "executionStatus": metadata.ExecutionStatusCancelled},
			},
		},
		{
			name:     "multiple statuses deduplicated",
			statuses: []metadata.JobStatus{metadata.JobStatusCompleted, metadata.JobStatusFailed},
			want: bson.A{
				bson.M{"dispatchStatus": metadata.DispatchStatusDispatched, "executionStatus": metadata.ExecutionStatusCompleted},
				bson.M{"dispatchStatus": metadata.DispatchStatusDispatched, "executionStatus": metadata.ExecutionStatusFailed},
			},
		},
		{
			name:     "duplicate statuses deduplicated",
			statuses: []metadata.JobStatus{metadata.JobStatusRunning, metadata.JobStatusRunning},
			want: bson.A{
				bson.M{"dispatchStatus": metadata.DispatchStatusDispatched, "executionStatus": metadata.ExecutionStatusRunning},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			query := buildListQuery(metadata.ListFilter{Statuses: tc.statuses})
			or, ok := query["$or"].(bson.A)
			if !ok {
				t.Fatalf("expected $or bson.A, got %T", query["$or"])
			}
			if len(or) != len(tc.want) {
				t.Fatalf("len $or = %d, want %d; got %+v", len(or), len(tc.want), or)
			}
			for i, cond := range or {
				if !reflect.DeepEqual(cond, tc.want[i]) {
					t.Fatalf("condition %d: got %+v, want %+v", i, cond, tc.want[i])
				}
			}
		})
	}
}

func TestBuildListQuery_NoStatuses(t *testing.T) {
	t.Parallel()
	query := buildListQuery(metadata.ListFilter{})
	if _, ok := query["$or"]; ok {
		t.Fatal("expected no $or when no statuses")
	}
}

func TestBuildListQuery_InvalidStatusIgnored(t *testing.T) {
	t.Parallel()
	bad := metadata.JobStatus("nope")
	query := buildListQuery(metadata.ListFilter{Statuses: []metadata.JobStatus{bad}})
	or, ok := query["$or"].(bson.A)
	if !ok {
		t.Fatalf("expected $or bson.A, got %T", query["$or"])
	}
	if len(or) != 0 {
		t.Fatalf("expected empty $or for invalid status, got %+v", or)
	}
}
