package main

import (
	"fmt"

	"github.com/daconjurer/jobby/internal/config"
)

func loadMigrateConfig() (config.MigrateConfig, error) {
	var mc config.MigrateConfig
	if err := config.LoadInto(&mc); err != nil {
		return config.MigrateConfig{}, fmt.Errorf("parsing migrate config: %w", err)
	}
	return mc, nil
}
