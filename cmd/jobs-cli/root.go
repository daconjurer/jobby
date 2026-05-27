package main

import (
	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/cmd/jobs-cli/commands"
	"github.com/spf13/cobra"
)

func newRootCmd(a *app.App) *cobra.Command {
	root := &cobra.Command{
		Use:   "jobs-cli",
		Short: "Operate on job metadata in MongoDB",
		Long: `Operate on job metadata stored in MongoDB.

Requires MONGODB_URI and related MONGODB_* variables (see .env.example).`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			format, err := cmd.Flags().GetString("output")
			if err != nil {
				return
			}
			a.Format = app.OutputFormat(format)
		},
	}

	root.PersistentFlags().String("output", string(app.OutputJSON), "Output format: json or table")

	root.AddCommand(
		commands.NewPingCmd(a),
		commands.NewGetCmd(a),
		commands.NewListCmd(a),
		commands.NewStatsCmd(a),
		commands.NewLogsCmd(a),
		commands.NewSeedCmd(a),
	)

	return root
}
