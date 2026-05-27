package app

import (
	"io"
	"os"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

type OutputFormat string

const (
	OutputJSON  OutputFormat = "json"
	OutputTable OutputFormat = "table"
)

// App holds shared CLI runtime state for subcommands.
type App struct {
	Service *service.MetadataService
	Writer  *metadata.MongoJobsWriter
	Out     io.Writer
	Format  OutputFormat
}

func New(svc *service.MetadataService, writer *metadata.MongoJobsWriter) *App {
	return &App{
		Service: svc,
		Writer:  writer,
		Out:     os.Stdout,
		Format:  OutputJSON,
	}
}
