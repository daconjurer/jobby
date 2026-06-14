package metadata

import "errors"

var (
	ErrJobNotFound        = errors.New("job not found")
	ErrEmptyUpdateJob     = errors.New("update job: patch has no fields to set")
	ErrJobAlreadyTerminal = errors.New("job already in terminal state")
)
