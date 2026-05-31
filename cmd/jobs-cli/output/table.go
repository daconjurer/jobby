package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

func WriteJobsTable(w io.Writer, jobs []metadata.JobMetadata) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "JOB ID\tNAME\tSTATUS\tPRIORITY\tRETRIES\tCREATED")
	for _, job := range jobs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\t%s\n",
			job.GetJobID(),
			job.GetName(),
			job.GetStatus(),
			job.GetPriority(),
			job.GetRetryCount(),
			formatTime(job.GetCreatedAt()),
		)
	}
	return tw.Flush()
}

func WriteStatsTable(w io.Writer, stats service.JobStats) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "STATUS\tCOUNT")
	fmt.Fprintf(tw, "pending\t%d\n", stats.Pending)
	fmt.Fprintf(tw, "running\t%d\n", stats.Running)
	fmt.Fprintf(tw, "completed\t%d\n", stats.Completed)
	fmt.Fprintf(tw, "failed\t%d\n", stats.Failed)
	fmt.Fprintf(tw, "cancelled\t%d\n", stats.Cancelled)
	fmt.Fprintf(tw, "total\t%d\n", stats.Total)
	return tw.Flush()
}

func WriteLogsTable(w io.Writer, logs []metadata.JobLog) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TIMESTAMP\tLEVEL\tMESSAGE\tSOURCE")
	for _, log := range logs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			formatTime(log.Timestamp),
			log.Level,
			truncate(log.Message, 80),
			log.Source,
		)
	}
	return tw.Flush()
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
