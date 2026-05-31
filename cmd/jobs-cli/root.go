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
		Long: `Operate on job metadata stored in MongoDB (parity with cmd/jobs-server HTTP API).

Requires MONGODB_URI and related MONGODB_* variables (see .env.example).

Subcommands: ping, create, get, list, stats, fail, cancel, retry, logs, seed.

Use --output json (default) for scripting, or --output table for human-readable list/stats/logs.`,
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
		commands.NewCreateCmd(a),
		commands.NewGetCmd(a),
		commands.NewListCmd(a),
		commands.NewStatsCmd(a),
		commands.NewFailCmd(a),
		commands.NewCancelCmd(a),
		commands.NewRetryCmd(a),
		commands.NewLogsCmd(a),
		commands.NewSeedCmd(a),
	)

	return root
}
