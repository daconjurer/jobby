package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/spf13/cobra"
)

type createInput struct {
	Name         string
	Priority     int
	PrioritySet  bool
	Tags         []string
	Payload      string
	PayloadFile  string
	Metadata     string
	MetadataFile string
}

func NewCreateCmd(a *app.App) *cobra.Command {
	var input createInput

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Enqueue a new job (metadata only)",
		Long: `Create a new job record in MongoDB. Does not dispatch work to workers.

For non-trivial JSON in shell scripts, prefer --payload-file and --metadata-file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			input.PrioritySet = cmd.Flags().Changed("priority")
			return RunCreate(cmd.Context(), a, input)
		},
	}

	cmd.Flags().StringVar(&input.Name, "name", "", "Job name (required)")
	cmd.Flags().IntVar(&input.Priority, "priority", 0, "Job priority (0-10)")
	cmd.Flags().StringArrayVar(&input.Tags, "tags", nil, "Job tag (repeatable)")
	cmd.Flags().StringVar(&input.Payload, "payload", "", "Job payload as a JSON object")
	cmd.Flags().StringVar(&input.PayloadFile, "payload-file", "", "Path to a JSON file containing the job payload")
	cmd.Flags().StringVar(&input.Metadata, "metadata", "", "Job metadata as a JSON object")
	cmd.Flags().StringVar(&input.MetadataFile, "metadata-file", "", "Path to a JSON file containing job metadata")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func RunCreate(ctx context.Context, a *app.App, input createInput) error {
	if input.Name == "" {
		return errors.New("name is required")
	}

	if input.PrioritySet && (input.Priority < 0 || input.Priority > 10) {
		return errors.New("priority must be between 0 and 10")
	}

	payload, err := parseJSONObject(input.Payload, input.PayloadFile, "payload")
	if err != nil {
		return err
	}

	meta, err := parseJSONObject(input.Metadata, input.MetadataFile, "metadata")
	if err != nil {
		return err
	}

	opts := service.CreateJobOptions{
		Tags:     input.Tags,
		Metadata: meta,
	}
	if input.PrioritySet {
		opts.Priority = &input.Priority
	}

	job, err := a.Service.CreateJob(ctx, input.Name, payload, opts)
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	return output.WriteJSON(a.Out, job)
}
