package seed

import "time"

// Options controls synthetic job and log generation.
type Options struct {
	Count         int
	LogsPerJobMin int
	LogsPerJobMax int
	WithLogs      bool
	Seed          int64
	BatchSize     int
	MaxAge        time.Duration
}

// Result reports how many documents were inserted.
type Result struct {
	JobsInserted int `json:"jobsInserted"`
	LogsInserted int `json:"logsInserted"`
}
