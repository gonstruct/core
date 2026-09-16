package config

import "time"

type Cors struct {
	AllowedMethods   []string
	AllowedOrigins   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	MaxAge           time.Duration
	AllowCredentials bool
}
