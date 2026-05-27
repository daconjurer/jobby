package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Validate checks Pulsar settings after env parsing.
func (c *PulsarConfig) Validate() error {
	var errs []error
	if strings.TrimSpace(c.SubscriptionName) == "" {
		errs = append(errs, fmt.Errorf(
			"PULSAR_SUBSCRIPTION_NAME must be non-empty: set a subscription name (default is jobber)",
		))
	}
	url := strings.TrimSpace(c.ServiceURL)
	if url != "" && !strings.HasPrefix(url, "pulsar://") && !strings.HasPrefix(url, "pulsar+ssl://") {
		errs = append(errs, fmt.Errorf(
			"PULSAR_SERVICE_URL (%q) must start with pulsar:// or pulsar+ssl://",
			c.ServiceURL,
		))
	}
	return errors.Join(errs...)
}

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
		return fmt.Errorf("APP_PORT must be a valid integer (got %q): set APP_PORT to a numeric listen port", c.Port)
	}
	if port < 1024 || port > 65535 {
		return fmt.Errorf(
			"APP_PORT (%d) must be between 1024 and 65535 (non-privileged range): choose a port your process may bind to",
			port,
		)
	}
	return nil
}
