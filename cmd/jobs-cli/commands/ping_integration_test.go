//go:build integration

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
)

func TestIntegration_Ping(t *testing.T) {
	cfg := integrationMongoConfig(t)
	ctx := context.Background()

	application, cleanup, err := app.Bootstrap(ctx, cfg)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	defer cleanup()

	var buf bytes.Buffer
	application.Out = &buf

	cmd := NewPingCmd(application)
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	var got healthResponse
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode health JSON: %v (body=%q)", err, buf.String())
	}
	if got.Status != "healthy" || got.Database != "connected" {
		t.Fatalf("health response: got %+v", got)
	}
}
