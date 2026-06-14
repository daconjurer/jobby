package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
)

func TestPingCommand_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	c := &cli.CLI{Out: &buf, Format: cli.OutputJSON}

	if err := WritePingHealth(c); err != nil {
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
