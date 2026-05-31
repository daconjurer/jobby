package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
)

func TestPingCommand_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	a := &app.App{Out: &buf, Format: app.OutputJSON}

	if err := WritePingHealth(a); err != nil {
		t.Fatalf("WritePingHealth: %v", err)
	}

	var got healthResponse
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode health JSON: %v", err)
	}
	if got.Status != "healthy" || got.Database != "connected" {
		t.Fatalf("health response: got %+v", got)
	}
}
