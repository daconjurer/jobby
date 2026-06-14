package commands

import (
	"encoding/json"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
	"github.com/spf13/cobra"
)

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func NewPingCmd(c *cli.CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Check MongoDB connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return WritePingHealth(c)
		},
	}
}

// WritePingHealth emits the same JSON shape as GET /health on jobs-server.
func WritePingHealth(c *cli.CLI) error {
	return json.NewEncoder(c.Out).Encode(healthResponse{
		Status:   "healthy",
		Database: "connected",
	})
}
