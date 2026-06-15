
package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
	"github.com/daconjurer/jobby/internal/jobs/appruntime"
	"github.com/daconjurer/jobby/internal/testutil"
)

func TestIntegration_Ping(t *testing.T) {
	cfg := integrationMongoConfig(t)
	ctx := context.Background()

	rt, cleanup, err := appruntime.Bootstrap(ctx, appruntime.Config{
		Mongo:            cfg,
		TopicsConfigPath: testutil.JobTopicsConfigPath(t),
	})
	if err != nil {
		t.Fatalf("appruntime.Bootstrap: %v", err)
	}
	defer cleanup()

	c := cli.New(rt.Metadata, rt.Enqueue, rt.Writer)

	var buf bytes.Buffer
	c.Out = &buf

	cmd := NewPingCmd(c)
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
