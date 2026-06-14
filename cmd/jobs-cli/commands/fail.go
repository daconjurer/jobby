package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/spf13/cobra"
)

type messageResponse struct {
	Message string `json:"message"`
}

func NewFailCmd(c *cli.CLI) *cobra.Command {
	var errMsg string

	cmd := &cobra.Command{
		Use:   "fail <jobId>",
		Short: "Mark a job as failed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunFail(cmd.Context(), c, args[0], errMsg)
		},
	}

	cmd.Flags().StringVarP(&errMsg, "error", "e", "", "Failure error message (required)")
	_ = cmd.MarkFlagRequired("error")

	return cmd
}

func RunFail(ctx context.Context, c *cli.CLI, jobID, errMsg string) error {
	if errMsg == "" {
		return errors.New("error is required")
	}

	if err := c.Service.FailJob(ctx, jobID, fmt.Errorf("%s", errMsg)); err != nil {
		return mapJobNotFound(err)
	}

	return output.WriteJSON(c.Out, messageResponse{Message: "job marked as failed"})
}
