package commands

import (
	"context"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/spf13/cobra"
)

func NewCancelCmd(c *cli.CLI) *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "cancel <jobId>",
		Short: "Cancel a non-terminal job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunCancel(cmd.Context(), c, args[0], reason)
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "Optional cancellation reason")

	return cmd
}

func RunCancel(ctx context.Context, c *cli.CLI, jobID, reason string) error {
	if err := c.Service.CancelJob(ctx, jobID, reason); err != nil {
		return mapJobNotFound(err)
	}

	return output.WriteJSON(c.Out, messageResponse{Message: "job cancelled"})
}
