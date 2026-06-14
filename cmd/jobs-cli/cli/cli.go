package cli

import (
	"io"
	"os"

	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

type OutputFormat string

const (
	OutputJSON  OutputFormat = "json"
	OutputTable OutputFormat = "table"
)

// CLI holds shared runtime state for jobs-cli subcommands.
type CLI struct {
	Service *service.MetadataService
	Enqueue *service.EnqueueService
	Writer  *mongodb.MongoJobsWriter
	Out     io.Writer
	Format  OutputFormat
}

// New constructs CLI state wired from appruntime.Bootstrap.
func New(svc *service.MetadataService, enqueue *service.EnqueueService, writer *mongodb.MongoJobsWriter) *CLI {
	return &CLI{
		Service: svc,
		Enqueue: enqueue,
		Writer:  writer,
		Out:     os.Stdout,
		Format:  OutputJSON,
	}
}
