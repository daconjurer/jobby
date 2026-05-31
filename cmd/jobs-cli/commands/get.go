package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/spf13/cobra"
)

func NewGetCmd(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "get <jobId>",
		Short: "Get a job by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunGet(cmd.Context(), a, args[0])
		},
	}
}

func RunGet(ctx context.Context, a *app.App, jobID string) error {
	job, err := a.Service.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, metadata.ErrJobNotFound) {
			return errors.New("job not found")
		}
		return fmt.Errorf("get job: %w", err)
	}

	return output.WriteJSON(a.Out, job)
}
