package commands

import (
	"context"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/spf13/cobra"
)

func NewCancelCmd(a *app.App) *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "cancel <jobId>",
		Short: "Cancel a non-terminal job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunCancel(cmd.Context(), a, args[0], reason)
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "Optional cancellation reason")

	return cmd
}

func RunCancel(ctx context.Context, a *app.App, jobID, reason string) error {
	if err := a.Service.CancelJob(ctx, jobID, reason); err != nil {
		return mapJobNotFound(err)
	}

	return output.WriteJSON(a.Out, messageResponse{Message: "job cancelled"})
}
