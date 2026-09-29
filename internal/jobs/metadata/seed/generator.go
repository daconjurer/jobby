package seed

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

var (
	defaultJobNames = []string{
		"process-kml",
		"export-report",
		"sync-inventory",
		"send-notification",
		"cleanup-storage",
		"import-csv",
		"rebuild-cache",
	}

	defaultTags = []string{
		"urgent",
		"batch",
		"retry",
		"nightly",
		"manual",
		"scheduled",
		"backfill",
	}

	logSources = []string{"worker", "service", "scheduler", "handler"}
)

type statusPairWeight struct {
	pair   metadata.CompositeStatus
	weight int
}

// statusDistribution weights produce the desired display-status distribution.
// The display status is computed from the CompositeStatus.
var statusDistribution = []statusPairWeight{
	{metadata.CompositeStatus{Dispatch: metadata.DispatchStatusPending, Execution: metadata.ExecutionStatusNotStarted}, 5},
	{metadata.CompositeStatus{Dispatch: metadata.DispatchStatusDispatched, Execution: metadata.ExecutionStatusNotStarted}, 3},
	{metadata.CompositeStatus{Dispatch: metadata.DispatchStatusFailed, Execution: metadata.ExecutionStatusNotStarted}, 2},
	{metadata.CompositeStatus{Dispatch: metadata.DispatchStatusDispatched, Execution: metadata.ExecutionStatusRunning}, 5},
	{metadata.CompositeStatus{Dispatch: metadata.DispatchStatusDispatched, Execution: metadata.ExecutionStatusCompleted}, 70},
	{metadata.CompositeStatus{Dispatch: metadata.DispatchStatusDispatched, Execution: metadata.ExecutionStatusFailed}, 10},
	{metadata.CompositeStatus{Dispatch: metadata.DispatchStatusDispatched, Execution: metadata.ExecutionStatusCancelled}, 5},
}

func newFaker(seed int64) *gofakeit.Faker {
	if seed != 0 {
		return gofakeit.New(uint64(seed))
	}
	return gofakeit.New(randomSeed())
}

func randomSeed() uint64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return uint64(time.Now().UnixNano())
	}
	return binary.LittleEndian.Uint64(b[:])
}

func pickCompositeStatus(f *gofakeit.Faker) metadata.CompositeStatus {
	total := 0
	for _, sw := range statusDistribution {
		total += sw.weight
	}
	n := f.IntRange(1, total)
	for _, sw := range statusDistribution {
		n -= sw.weight
		if n <= 0 {
			return sw.pair
		}
	}
	return metadata.CompositeStatus{Dispatch: metadata.DispatchStatusDispatched, Execution: metadata.ExecutionStatusCompleted}
}

func randomPayload(f *gofakeit.Faker) map[string]any {
	size := f.IntRange(1, 4)
	payload := make(map[string]any, size)
	for i := 0; i < size; i++ {
		key := f.Word()
		switch f.IntRange(0, 2) {
		case 0:
			payload[key] = f.Sentence(f.IntRange(1, 5))
		case 1:
			payload[key] = f.IntRange(1, 10_000)
		default:
			payload[key] = f.Bool()
		}
	}
	return payload
}

func randomMetadata(f *gofakeit.Faker) map[string]any {
	return map[string]any{
		"source":   f.RandomString(logSources),
		"workerId": f.UUID(),
		"version":  fmt.Sprintf("%d.%d.%d", f.IntRange(0, 3), f.IntRange(0, 9), f.IntRange(0, 20)),
	}
}

func randomTags(f *gofakeit.Faker) []string {
	count := f.IntRange(0, 3)
	if count == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, count)
	tags := make([]string, 0, count)
	for len(tags) < count {
		tag := f.RandomString(defaultTags)
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	return tags
}

func randomCreatedAt(f *gofakeit.Faker, maxAge time.Duration) time.Time {
	if maxAge <= 0 {
		maxAge = 30 * 24 * time.Hour
	}
	start := time.Now().UTC().Add(-maxAge)
	return f.DateRange(start, time.Now().UTC())
}

func fakeTopic(f *gofakeit.Faker) string {
	return fmt.Sprintf("persistent://public/default/seed/%s", f.Word())
}

func applyCompositeStatus(job *metadata.JobMetadataModel, pair metadata.CompositeStatus, f *gofakeit.Faker) {
	job.DispatchStatus = pair.Dispatch
	job.ExecutionStatus = pair.Execution
	job.Status = pair.DisplayStatus()

	switch pair.Dispatch {
	case metadata.DispatchStatusPending:
		job.Topic = fakeTopic(f)
	case metadata.DispatchStatusDispatched:
		job.Topic = fakeTopic(f)
		dispatched := job.CreatedAt.Add(randomDuration(f, 1*time.Second, 30*time.Minute))
		job.DispatchedAt = &dispatched
	case metadata.DispatchStatusFailed:
		job.Topic = fakeTopic(f)
		job.DispatchAttempts = f.IntRange(1, 5)
		job.DispatchLastError = f.Sentence(f.IntRange(3, 8))
	}

	switch pair.Execution {
	case metadata.ExecutionStatusRunning:
		started := job.CreatedAt.Add(randomDuration(f, 1*time.Minute, 2*time.Hour))
		job.StartedAt = &started
	case metadata.ExecutionStatusCompleted, metadata.ExecutionStatusCancelled:
		started := job.CreatedAt.Add(randomDuration(f, 1*time.Minute, 1*time.Hour))
		completed := started.Add(randomDuration(f, 1*time.Second, 30*time.Minute))
		job.StartedAt = &started
		job.CompletedAt = &completed
	case metadata.ExecutionStatusFailed:
		if f.Bool() {
			started := job.CreatedAt.Add(randomDuration(f, 1*time.Minute, 1*time.Hour))
			completed := started.Add(randomDuration(f, 1*time.Second, 30*time.Minute))
			job.StartedAt = &started
			job.CompletedAt = &completed
		} else {
			completed := job.CreatedAt.Add(randomDuration(f, 1*time.Minute, 2*time.Hour))
			job.CompletedAt = &completed
		}
		errorMsg := f.Sentence(f.IntRange(3, 12))
		job.Errors = []metadata.JobError{
			{
				Type:         metadata.JobErrorTypeExecution,
				RetryAttempt: job.RetryCount,
				Error:        errorMsg,
				Timestamp:    *job.CompletedAt,
			},
		}
	}
}

func randomDuration(f *gofakeit.Faker, min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	seconds := f.IntRange(int(min.Seconds()), int(max.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	return time.Duration(seconds) * time.Second
}

func randomTimestampBetween(f *gofakeit.Faker, start, end time.Time) time.Time {
	if !end.After(start) {
		return start
	}
	return f.DateRange(start, end)
}

// BuildJob generates one job record that passes metadata validation.
func BuildJob(f *gofakeit.Faker, maxAge time.Duration) (*metadata.JobMetadataModel, error) {
	job := metadata.NewJobMetadata(
		metadata.GenerateJobID(),
		f.RandomString(defaultJobNames),
		randomPayload(f),
	)

	job.CreatedAt = randomCreatedAt(f, maxAge)
	job.Priority = f.IntRange(0, 10)
	job.Tags = randomTags(f)
	job.Metadata = randomMetadata(f)

	pair := pickCompositeStatus(f)
	applyCompositeStatus(job, pair, f)

	if pair.Execution == metadata.ExecutionStatusFailed || pair.Execution == metadata.ExecutionStatusCancelled {
		job.RetryCount = f.IntRange(0, 3)
	} else {
		job.RetryCount = f.IntRange(0, 1)
	}

	if err := job.Validate(); err != nil {
		return nil, fmt.Errorf("generated invalid job: %w", err)
	}
	return job, nil
}

// BuildLogs generates log entries for a job within its lifecycle window.
func BuildLogs(job *metadata.JobMetadataModel, min, max int, f *gofakeit.Faker) []metadata.JobLog {
	if max < min {
		max = min
	}
	if max == 0 {
		return nil
	}

	count := f.IntRange(min, max)
	if count == 0 {
		return nil
	}

	end := time.Now().UTC()
	if job.CompletedAt != nil {
		end = *job.CompletedAt
	} else if job.StartedAt != nil {
		end = *job.StartedAt
	}

	logs := make([]metadata.JobLog, 0, count)
	for i := 0; i < count; i++ {
		level := f.RandomString([]string{
			string(metadata.LogLevelDebug),
			string(metadata.LogLevelInfo),
			string(metadata.LogLevelInfo),
			string(metadata.LogLevelInfo),
			string(metadata.LogLevelWarn),
			string(metadata.LogLevelError),
		})

		logEntry := metadata.NewJobLogWithContext(
			job.JobID,
			metadata.LogLevel(level),
			f.Sentence(f.IntRange(3, 10)),
			map[string]any{
				"step":     f.IntRange(1, count),
				"attempt":  job.RetryCount + 1,
				"duration": f.IntRange(10, 5000),
			},
		)
		logEntry.Timestamp = randomTimestampBetween(f, job.CreatedAt, end)
		logEntry.Source = f.RandomString(logSources)
		if logEntry.Level == metadata.LogLevelError || logEntry.Level == metadata.LogLevelFatal {
			logEntry.StackTrace = f.Sentence(f.IntRange(5, 15))
		}
		logs = append(logs, logEntry)
	}
	return logs
}
