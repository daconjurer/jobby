package commands

import (
	"fmt"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

// BuildListFilter constructs a ListFilter from CLI flag values (parity with ListJobs handler).
func BuildListFilter(limit, skip int, sortBy string, sortDesc bool, status string, tags, names []string) (metadata.ListFilter, error) {
	filter := metadata.ListFilter{
		Limit:    limit,
		Skip:     skip,
		SortBy:   sortBy,
		SortDesc: sortDesc,
	}

	if status != "" {
		st := metadata.JobStatus(status)
		if !st.IsValid() {
			return filter, fmt.Errorf("invalid status: %q", status)
		}
		filter.Statuses = []metadata.JobStatus{st}
	}

	if len(tags) > 0 {
		filter.Tags = tags
	}
	if len(names) > 0 {
		filter.Names = names
	}

	return filter, nil
}

// BuildLogFilter constructs a LogFilter from CLI flag values (parity with GetJobLogs handler).
func BuildLogFilter(limit, skip int, levels []string) (metadata.LogFilter, error) {
	filter := metadata.LogFilter{
		Limit: limit,
		Skip:  skip,
	}

	if len(levels) == 0 {
		return filter, nil
	}

	filter.Levels = make([]metadata.LogLevel, len(levels))
	for i, l := range levels {
		lvl := metadata.LogLevel(l)
		if !lvl.IsValid() {
			return filter, fmt.Errorf("invalid log level: %q", l)
		}
		filter.Levels[i] = lvl
	}

	return filter, nil
}
