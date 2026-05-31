package commands

import (
	"errors"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

func mapJobNotFound(err error) error {
	if errors.Is(err, metadata.ErrJobNotFound) {
		return errors.New("job not found")
	}
	return err
}
