// Package config loads domain-specific configuration from environment variables
// using struct tags interpreted by github.com/caarlos0/env/v11.
//
// Load configuration with [LoadInto] or [LoadIntoWithOptions] into a struct such as
// [MongoConfig] or [ServerConfig], then call [MongoConfig.Validate] / [ServerConfig.Validate]
// before using the values.
package config
