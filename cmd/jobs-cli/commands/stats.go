package commands

import (
	"context"
	"fmt"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/spf13/cobra"
)

func NewStatsCmd(c *cli.CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show job counts by status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunStats(cmd.Context(), c)
		},
	}
}

func RunStats(ctx context.Context, c *cli.CLI) error {
	stats, err := c.Service.GetJobStats(ctx)
	if err != nil {
		return fmt.Errorf("job stats: %w", err)
	}

	if c.Format == cli.OutputTable {
		return output.WriteStatsTable(c.Out, stats)
	}

	return output.WriteJSON(c.Out, stats)
}
