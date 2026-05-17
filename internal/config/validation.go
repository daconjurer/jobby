package config

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Validate checks MongoDB pool and timeout constraints after env parsing.
func (c *MongoConfig) Validate() error {
	var errs []error
	if c.MaxPoolSize < c.MinPoolSize {
		errs = append(errs, fmt.Errorf(
			"MONGODB_MAX_POOL_SIZE (%d) must be >= MONGODB_MIN_POOL_SIZE (%d): raise max, lower min, or align both with your cluster capacity",
			c.MaxPoolSize, c.MinPoolSize,
		))
	}
	if c.Timeout < time.Second {
		errs = append(errs, fmt.Errorf(
			"MONGODB_TIMEOUT (%s) must be at least 1s (e.g. 1s, 10s): short timeouts cause flaky connects",
			c.Timeout,
		))
	}
	if c.MaxPoolSize > 1000 {
		errs = append(errs, fmt.Errorf(
			"MONGODB_MAX_POOL_SIZE (%d) exceeds the allowed maximum (1000): reduce to stay within driver and server limits",
			c.MaxPoolSize,
		))
	}
	return errors.Join(errs...)
}

// Validate checks listen port constraints after env parsing.
func (c *ServerConfig) Validate() error {
	port, err := strconv.Atoi(c.Port)
	if err != nil {
		return fmt.Errorf("PORT must be a valid integer (got %q): set PORT to a numeric listen port", c.Port)
	}
	if port < 1024 || port > 65535 {
		return fmt.Errorf(
			"PORT (%d) must be between 1024 and 65535 (non-privileged range): choose a port your process may bind to",
			port,
		)
	}
	return nil
}
