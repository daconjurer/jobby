package mongodb

import (
	"testing"

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
