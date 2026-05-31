package commands

import (
	"context"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/spf13/cobra"
)

func NewRetryCmd(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "retry <jobId>",
		Short: "Retry a failed job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunRetry(cmd.Context(), a, args[0])
		},
	}
}

func RunRetry(ctx context.Context, a *app.App, jobID string) error {
	if err := a.Service.RetryJob(ctx, jobID); err != nil {
		return mapJobNotFound(err)
	}

	return output.WriteJSON(a.Out, messageResponse{Message: "job retry initiated"})
}
