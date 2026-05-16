package settings

import (
	"fmt"
	"time"
)

func ParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

func ParseUint64(s string) uint64 {
	var v uint64
	_, _ = fmt.Sscanf(s, "%d", &v)
	return v
}
