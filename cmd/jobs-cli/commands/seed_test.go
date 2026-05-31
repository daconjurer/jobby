package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/internal/jobs/metadata/seed"
)

func TestSeedCommand_MissingWriter(t *testing.T) {
	a := app.New(nil, nil)
	a.Out = &bytes.Buffer{}

	err := RunSeed(t.Context(), a, seed.Options{Count: 1})
	if err == nil {
		t.Fatal("expected error when writer is nil")
	}
}

func TestSeedCommand_InvalidCount(t *testing.T) {
	a := app.New(nil, nil)
	a.Out = &bytes.Buffer{}

	err := RunSeed(t.Context(), a, seed.Options{Count: 0})
	if err == nil {
		t.Fatal("expected error for count 0")
	}
}

func TestSeedCommand_JSONOutputShape(t *testing.T) {
	var result seed.Result
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]int
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := decoded["jobsInserted"]; !ok {
		t.Fatal("expected jobsInserted field")
	}
	if _, ok := decoded["logsInserted"]; !ok {
		t.Fatal("expected logsInserted field")
	}
}
