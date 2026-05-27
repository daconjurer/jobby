package app

import (
	"io"
	"os"

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
	Out     io.Writer
	Format  OutputFormat
}

func New(svc *service.MetadataService) *App {
	return &App{
		Service: svc,
		Out:     os.Stdout,
		Format:  OutputJSON,
	}
}
