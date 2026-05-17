// Package config loads domain-specific configuration from environment variables
// using struct tags interpreted by github.com/caarlos0/env/v11.
//
// Load configuration with [LoadInto] into a concrete struct (e.g. [MongoConfig], [ServerConfig]).
package config
