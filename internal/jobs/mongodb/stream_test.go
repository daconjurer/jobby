package mongodb

import (
	"context"
	"errors"
	"testing"

	"github.com/daconjurer/jobby/internal/jobs/dispatch"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestInsertPendingDispatchPipeline_MatchesInsertPendingDispatchOnly(t *testing.T) {
	if len(insertPendingDispatchPipeline) != 1 {
		t.Fatalf("pipeline len=%d want 1", len(insertPendingDispatchPipeline))
	}

	stage := insertPendingDispatchPipeline[0]
	if stage[0].Key != "$match" {
		t.Fatalf("stage key=%q want $match", stage[0].Key)
	}

	match, ok := stage[0].Value.(bson.D)
	if !ok {
		t.Fatalf("match value type=%T", stage[0].Value)
	}

	want := map[string]any{
		"operationType":       "insert",
		"fullDocument.status": metadata.JobStatusPendingDispatch,
	}
	got := make(map[string]any, len(match))
	for _, elem := range match {
		got[elem.Key] = elem.Value
	}
	for key, wantVal := range want {
		if got[key] != wantVal {
			t.Fatalf("%s=%v want %v", key, got[key], wantVal)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("match keys=%v want %v", got, want)
	}
}

type recordingDispatchHandler struct {
	handled []dispatch.JobDispatchProjection
	err     error
}

func (h *recordingDispatchHandler) HandleDispatch(_ context.Context, job dispatch.JobDispatchProjection) error {
	h.handled = append(h.handled, job)
	return h.err
}

func TestNewStreamWatcher_NilTokenStoreUsesNop(t *testing.T) {
	watcher := NewStreamWatcher(nil, &recordingDispatchHandler{}, nil)
	if watcher.tokens == nil {
		t.Fatal("expected non-nil token store")
	}
	if _, ok := watcher.tokens.(NopResumeTokenStore); !ok {
		t.Fatalf("token store type=%T want NopResumeTokenStore", watcher.tokens)
	}
}

func TestHandlePendingDispatchInsert_DispatchesJobWithTopic(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata("job-1", "load-products", map[string]any{"k": "v"})
	job.Topic = "persistent://public/default/jobs"
	handler := &recordingDispatchHandler{}

	if err := handlePendingDispatchInsert(ctx, handler, job); err != nil {
		t.Fatal(err)
	}
	if len(handler.handled) != 1 {
		t.Fatalf("handled=%d want 1", len(handler.handled))
	}
	got := handler.handled[0]
	if got.JobID != job.JobID || got.Name != job.Name || got.Topic != job.Topic {
		t.Fatalf("handled=%+v", got)
	}
	if got.Payload["k"] != "v" {
		t.Fatalf("payload=%v", got.Payload)
	}
}

func TestHandlePendingDispatchInsert_MissingTopic(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata("job-1", "load-products", nil)
	handler := &recordingDispatchHandler{}

	err := handlePendingDispatchInsert(ctx, handler, job)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(handler.handled) != 0 {
		t.Fatalf("handled=%v want none", handler.handled)
	}
}

func TestHandlePendingDispatchInsert_HandlerErrorPropagates(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata("job-1", "load-products", nil)
	job.Topic = "topic-a"
	wantErr := errors.New("dispatch failed")
	handler := &recordingDispatchHandler{err: wantErr}

	err := handlePendingDispatchInsert(ctx, handler, job)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err=%v want %v", err, wantErr)
	}
}
