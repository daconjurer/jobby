package commands

import (
	"context"
	"fmt"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/cmd/jobs-cli/output"
	"github.com/spf13/cobra"
)

type listResponse struct {
	Jobs  any `json:"jobs"`
	Count int `json:"count"`
}

func NewListCmd(a *app.App) *cobra.Command {
	var (
		limit    int
		skip     int
		sortBy   string
		sortDesc bool
		sortAsc  bool
		status   string
		tags     []string
		names    []string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List jobs with optional filters",
		RunE: func(cmd *cobra.Command, args []string) error {
			desc := sortDesc
			if sortAsc {
				desc = false
			}
			return RunList(cmd.Context(), a, limit, skip, sortBy, desc, status, tags, names)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of jobs to return")
	cmd.Flags().IntVar(&skip, "skip", 0, "Number of jobs to skip")
	cmd.Flags().StringVar(&sortBy, "sort-by", "createdAt", "Field to sort by")
	cmd.Flags().BoolVar(&sortDesc, "sort-desc", true, "Sort descending")
	cmd.Flags().BoolVar(&sortAsc, "sort-asc", false, "Sort ascending (overrides --sort-desc)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by job status")
	cmd.Flags().StringArrayVar(&tags, "tags", nil, "Filter by tag (repeatable)")
	cmd.Flags().StringArrayVar(&names, "names", nil, "Filter by job name (repeatable)")

	return cmd
}

func RunList(ctx context.Context, a *app.App, limit, skip int, sortBy string, sortDesc bool, status string, tags, names []string) error {
	filter, err := BuildListFilter(limit, skip, sortBy, sortDesc, status, tags, names)
	if err != nil {
		return err
	}

	jobs, err := a.Service.ListJobs(ctx, filter)
	if err != nil {
		return fmt.Errorf("list jobs: %w", err)
	}

	if a.Format == app.OutputTable {
		return output.WriteJobsTable(a.Out, jobs)
	}

	return output.WriteJSON(a.Out, listResponse{
		Jobs:  jobs,
		Count: len(jobs),
	})
}
