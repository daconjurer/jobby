package config

import (
	"errors"

	"github.com/caarlos0/env/v11"
)

// ErrNilDestination is returned when [LoadInto] is called with a nil destination pointer.
var ErrNilDestination = errors.New("config: nil destination")

// LoadInto parses the current process environment into destination using env tags on T.
func LoadInto[T any](destination *T) error {
	if destination == nil {
		return ErrNilDestination
	}
	return env.Parse(destination)
}
