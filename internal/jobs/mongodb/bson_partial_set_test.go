package mongodb

import (
	"reflect"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func ptrStatus(s metadata.JobStatus) *metadata.JobStatus { return &s }

func TestUpdateJob_BSONReflectLayout(t *testing.T) {
	modelType := reflect.TypeFor[metadata.JobMetadataModel]()
	jobType := reflect.TypeFor[metadata.UpdateJob]()

	for i := range jobType.NumField() {
		sf := jobType.Field(i)
		if !sf.IsExported() {
			continue
		}
		t.Run(sf.Name, func(t *testing.T) {
			if sf.Type.Kind() != reflect.Pointer {
				t.Fatalf("field kind = %v, want pointer type for omit semantics", sf.Type.Kind())
			}
			tag := sf.Tag.Get("bson")
			key, ok := bsonNameFromTag(tag)
			if !ok {
				t.Fatal("exported field requires a bson document name tag (must not be '-' or empty)")
			}
			modelField, found := modelType.FieldByName(sf.Name)
			if !found {
				t.Fatalf("no matching field on JobMetadataModel named %s", sf.Name)
			}
			modelKey, mok := bsonNameFromTag(modelField.Tag.Get("bson"))
			if !mok {
				t.Fatalf("JobMetadataModel.%s missing bson tag", sf.Name)
			}
			if key != modelKey {
				t.Fatalf("bson key %q differs from JobMetadataModel (%q)", key, modelKey)
			}
		})
	}
}

func TestBSONPartialSet_UpdateJob(t *testing.T) {
	t.Parallel()

	when := time.Unix(1_700_000_000, 0).UTC()
	name := "n"
	pr := 3
	em := ""

	t.Run("omits_nil_pointers", func(t *testing.T) {
		t.Parallel()
		m, err := bsonPartialSet(&metadata.UpdateJob{})
		if err != nil {
			t.Fatal(err)
		}
		if len(m) != 0 {
			t.Fatalf("got %v, want empty bson.M", m)
		}
	})

	t.Run("subset", func(t *testing.T) {
		t.Parallel()
		st := metadata.JobStatusRunning
		m, err := bsonPartialSet(&metadata.UpdateJob{Status: &st, StartedAt: &when})
		if err != nil {
			t.Fatal(err)
		}
		if len(m) != 2 || m["status"] != metadata.JobStatusRunning || !m["startedAt"].(time.Time).Equal(when) {
			t.Fatalf("got %+v", m)
		}
	})

	t.Run("full_round_trip_shapes", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"k": 1}
		meta := map[string]any{"a": "b"}
		tags := []string{"x"}
		m, err := bsonPartialSet(&metadata.UpdateJob{
			Status:      ptrStatus(metadata.JobStatusFailed),
			Name:        &name,
			Priority:    &pr,
			StartedAt:   &when,
			CompletedAt: &when,
			Payload:     &payload,
			Metadata:    &meta,
			Error:       &em,
			Tags:        &tags,
		})
		if err != nil {
			t.Fatal(err)
		}
		want := bson.M{
			"status":      metadata.JobStatusFailed,
			"name":        name,
			"priority":    pr,
			"startedAt":   when,
			"completedAt": when,
			"payload":     payload,
			"metadata":    meta,
			"error":       em,
			"tags":        tags,
		}
		if len(m) != len(want) {
			t.Fatalf("len %d vs %d, got %+v want %+v", len(m), len(want), m, want)
		}
		for k, vw := range want {
			got, ok := m[k]
			if !ok {
				t.Fatalf("missing key %q", k)
			}
			if !reflect.DeepEqual(got, vw) {
				t.Fatalf("key %q: %#v vs %#v", k, got, vw)
			}
		}
	})
}

// TestBSONPartialSet_WithNestedStruct documents one-level reflection: nullable fields can point to structs.
// bsonPartialSet emits the dereferenced nested value as one $set value; it does not walk inner pointer fields recursively.
func TestBSONPartialSet_WithNestedStruct(t *testing.T) {
	t.Parallel()

	type innerBench struct {
		Score *int    `bson:"score,omitempty"`
		Note  *string `bson:"note,omitempty"`
	}

	type outerBench struct {
		Topic *string     `bson:"topic,omitempty"`
		Extra *innerBench `bson:"extra,omitempty"`
	}

	t.Run("whole_nested_subdocument", func(t *testing.T) {
		t.Parallel()

		score := 42
		note := "nested"
		topic := "top"
		innerVal := innerBench{Score: &score, Note: &note}

		got, err := bsonPartialSet(&outerBench{Topic: &topic, Extra: &innerVal})
		if err != nil {
			t.Fatal(err)
		}

		want := bson.M{
			"topic": topic,
			"extra": innerVal,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("deep equal failed\ngot:  %+v\nwant: %+v", got, want)
		}
	})

	t.Run("omit_nil_nested_pointer", func(t *testing.T) {
		t.Parallel()

		topic := "only-topic"
		got, err := bsonPartialSet(&outerBench{Topic: &topic})
		if err != nil {
			t.Fatal(err)
		}

		want := bson.M{"topic": topic}
		if _, hasExtra := got["extra"]; hasExtra {
			t.Fatalf("nested extra omitted when nil pointer; got %#v", got)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v want %#v", got, want)
		}
	})
}
