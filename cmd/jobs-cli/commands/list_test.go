package commands

import (
	"reflect"
	"strings"
	"testing"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

func TestListCommand_InvalidStatus(t *testing.T) {
	_, err := BuildListFilter(50, 0, "createdAt", true, "nope", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	if !strings.Contains(err.Error(), "invalid status") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListCommand_BuildsFilter(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		skip     int
		sortBy   string
		sortDesc bool
		status   string
		tags     []string
		names    []string
		want     metadata.ListFilter
		wantErr  bool
	}{
		{
			name:     "defaults",
			limit:    50,
			skip:     0,
			sortBy:   "createdAt",
			sortDesc: true,
			want: metadata.ListFilter{
				Limit:    50,
				Skip:     0,
				SortBy:   "createdAt",
				SortDesc: true,
			},
		},
		{
			name:     "status and tags",
			limit:    10,
			skip:     5,
			sortBy:   "priority",
			sortDesc: false,
			status:   "pending_dispatch",
			tags:     []string{"a", "b"},
			names:    []string{"job-a"},
			want: metadata.ListFilter{
				Limit:    10,
				Skip:     5,
				SortBy:   "priority",
				SortDesc: false,
				Statuses: []metadata.JobStatus{metadata.JobStatusPendingDispatch},
				Tags:     []string{"a", "b"},
				Names:    []string{"job-a"},
			},
		},
		{
			name:    "invalid status",
			status:  "bad",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildListFilter(tt.limit, tt.skip, tt.sortBy, tt.sortDesc, tt.status, tt.tags, tt.names)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildListFilter: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("filter mismatch\nwant %+v\ngot  %+v", tt.want, got)
			}
		})
	}
}

func TestLogsCommand_InvalidLevel(t *testing.T) {
	_, err := BuildLogFilter(100, 0, []string{"nope"})
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
	if !strings.Contains(err.Error(), "invalid log level") {
		t.Fatalf("unexpected error: %v", err)
	}
}
