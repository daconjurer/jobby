package commands

import (
	"context"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/spf13/cobra"
)

func NewRetryCmd(c *cli.CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "retry <jobId>",
		Short: "Retry a failed job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunRetry(cmd.Context(), c, args[0])
		},
	}
}

func RunRetry(ctx context.Context, c *cli.CLI, jobID string) error {
	if err := c.Service.RetryJob(ctx, jobID); err != nil {
		return mapJobNotFound(err)
	}

	return output.WriteJSON(c.Out, messageResponse{Message: "job retry initiated"})
}
