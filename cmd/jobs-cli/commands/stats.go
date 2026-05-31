package commands

import (
	"context"
	"fmt"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/spf13/cobra"
)

func NewStatsCmd(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show job counts by status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunStats(cmd.Context(), a)
		},
	}
}

func RunStats(ctx context.Context, a *app.App) error {
	stats, err := a.Service.GetJobStats(ctx)
	if err != nil {
		return fmt.Errorf("job stats: %w", err)
	}

	return output.WriteJSON(a.Out, stats)
}
