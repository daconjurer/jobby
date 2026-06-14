package main

import (
	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
	"github.com/daconjurer/jobby/cmd/jobs-cli/commands"
	"github.com/spf13/cobra"
)

func newRootCmd(c *cli.CLI) *cobra.Command {
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
			c.Format = cli.OutputFormat(format)
		},
	}

	root.PersistentFlags().String("output", string(cli.OutputJSON), "Output format: json or table")

	root.AddCommand(
		commands.NewPingCmd(c),
		commands.NewCreateCmd(c),
		commands.NewGetCmd(c),
		commands.NewListCmd(c),
		commands.NewStatsCmd(c),
		commands.NewFailCmd(c),
		commands.NewCancelCmd(c),
		commands.NewRetryCmd(c),
		commands.NewLogsCmd(c),
		commands.NewSeedCmd(c),
	)

	return root
}
