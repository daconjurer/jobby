package settings

import (
	"fmt"
	"time"
)

// ParseDuration parses s as a [time.Duration]. If parsing fails it returns 10s.
//
// Deprecated: Load duration-typed fields with internal/config and github.com/caarlos0/env/v11 instead.
func ParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

// ParseUint64 parses an unsigned integer from s. On failure it returns 0 (see [fmt.Sscanf]).
//
// Deprecated: Use internal/config instead.
func ParseUint64(s string) uint64 {
	var v uint64
	_, _ = fmt.Sscanf(s, "%d", &v)
	return v
}
