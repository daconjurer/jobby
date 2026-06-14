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
	if err := writeLine(tw, "JOB ID\tNAME\tSTATUS\tPRIORITY\tRETRIES\tCREATED"); err != nil {
		return err
	}
	for _, job := range jobs {
		if err := writeFormat(tw, "%s\t%s\t%s\t%d\t%d\t%s\n",
			job.GetJobID(),
			job.GetName(),
			job.GetStatus(),
			job.GetPriority(),
			job.GetRetryCount(),
			formatTime(job.GetCreatedAt()),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func WriteStatsTable(w io.Writer, stats service.JobStats) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if err := writeLine(tw, "STATUS\tCOUNT"); err != nil {
		return err
	}
	if err := writeFormat(tw, "pending_dispatch\t%d\n", stats.PendingDispatch); err != nil {
		return err
	}
	if err := writeFormat(tw, "dispatched\t%d\n", stats.Dispatched); err != nil {
		return err
	}
	if err := writeFormat(tw, "dispatch_failed\t%d\n", stats.DispatchFailed); err != nil {
		return err
	}
	if err := writeFormat(tw, "running\t%d\n", stats.Running); err != nil {
		return err
	}
	if err := writeFormat(tw, "completed\t%d\n", stats.Completed); err != nil {
		return err
	}
	if err := writeFormat(tw, "failed\t%d\n", stats.Failed); err != nil {
		return err
	}
	if err := writeFormat(tw, "cancelled\t%d\n", stats.Cancelled); err != nil {
		return err
	}
	if err := writeFormat(tw, "total\t%d\n", stats.Total); err != nil {
		return err
	}
	return tw.Flush()
}

func WriteLogsTable(w io.Writer, logs []metadata.JobLog) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if err := writeLine(tw, "TIMESTAMP\tLEVEL\tMESSAGE\tSOURCE"); err != nil {
		return err
	}
	for _, log := range logs {
		if err := writeFormat(tw, "%s\t%s\t%s\t%s\n",
			formatTime(log.Timestamp),
			log.Level,
			truncate(log.Message, 80),
			log.Source,
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeLine(w io.Writer, args ...any) error {
	_, err := fmt.Fprintln(w, args...)
	return err
}

func writeFormat(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
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
