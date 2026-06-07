package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/spf13/cobra"
)

func NewGetCmd(c *cli.CLI) *cobra.Command {
	return &cobra.Command{
		Use:   "get <jobId>",
		Short: "Get a job by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunGet(cmd.Context(), c, args[0])
		},
	}
}

func RunGet(ctx context.Context, c *cli.CLI, jobID string) error {
	job, err := c.Service.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, metadata.ErrJobNotFound) {
			return errors.New("job not found")
		}
		return fmt.Errorf("get job: %w", err)
	}

	return output.WriteJSON(c.Out, job)
}
