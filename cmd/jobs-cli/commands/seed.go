package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/daconjurer/jobby/internal/jobs/metadata/seed"
	"github.com/spf13/cobra"
)

func NewSeedCmd(a *app.App) *cobra.Command {
	var (
		count         int
		logsPerJobMin int
		logsPerJobMax int
		withLogs      bool
		seedValue     int64
		batchSize     int
		maxAge        time.Duration
	)

	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Insert random valid job data for testing",
		Long: `Generate and insert random but schema-valid job metadata (and optional logs).

Job records respect domain validation rules: valid UUIDs, status-specific timestamps,
priority bounds, and failed jobs include error messages.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunSeed(cmd.Context(), a, seed.Options{
				Count:         count,
				LogsPerJobMin: logsPerJobMin,
				LogsPerJobMax: logsPerJobMax,
				WithLogs:      withLogs,
				Seed:          seedValue,
				BatchSize:     batchSize,
				MaxAge:        maxAge,
			})
		},
	}

	cmd.Flags().IntVar(&count, "count", 100, "Number of jobs to insert")
	cmd.Flags().IntVar(&logsPerJobMin, "logs-per-job-min", 1, "Minimum logs per job when --with-logs is set")
	cmd.Flags().IntVar(&logsPerJobMax, "logs-per-job-max", 5, "Maximum logs per job when --with-logs is set")
	cmd.Flags().BoolVar(&withLogs, "with-logs", true, "Also insert log entries for each seeded job")
	cmd.Flags().Int64Var(&seedValue, "seed", 0, "RNG seed for reproducible data (0 = non-deterministic)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 500, "MongoDB insert batch size")
	cmd.Flags().DurationVar(&maxAge, "max-age", 30*24*time.Hour, "Maximum age of generated createdAt timestamps")

	return cmd
}

func RunSeed(ctx context.Context, a *app.App, opts seed.Options) error {
	if a.Writer == nil {
		return fmt.Errorf("mongo writer is not configured")
	}

	result, err := seed.Run(ctx, a.Writer, opts)
	if err != nil {
		return fmt.Errorf("seed database: %w", err)
	}

	return output.WriteJSON(a.Out, result)
}
