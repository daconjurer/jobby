package commands

import (
	"context"
	"fmt"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/spf13/cobra"
)

type logsResponse struct {
	Logs  any `json:"logs"`
	Count int `json:"count"`
}

func NewLogsCmd(a *app.App) *cobra.Command {
	var (
		limit  int
		skip   int
		levels []string
	)

	cmd := &cobra.Command{
		Use:   "logs <jobId>",
		Short: "Get logs for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunLogs(cmd.Context(), a, args[0], limit, skip, levels)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of log entries to return")
	cmd.Flags().IntVar(&skip, "skip", 0, "Number of log entries to skip")
	cmd.Flags().StringArrayVar(&levels, "levels", nil, "Filter by log level (repeatable)")

	return cmd
}

func RunLogs(ctx context.Context, a *app.App, jobID string, limit, skip int, levels []string) error {
	filter, err := BuildLogFilter(limit, skip, levels)
	if err != nil {
		return err
	}

	logs, err := a.Service.GetJobLogs(ctx, jobID, filter)
	if err != nil {
		return fmt.Errorf("get job logs: %w", err)
	}

	return output.WriteJSON(a.Out, logsResponse{
		Logs:  logs,
		Count: len(logs),
	})
}
