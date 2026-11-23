package config

import "time"

// Server is how the HTTP server listens, handed to routing.WithConfig.
type Server struct {
	// Host is the address it listens on: 0.0.0.0 when empty.
	Host string
	// Port is the port it listens on: 80 when 0.
	Port int
	// ShutdownTimeout is how long a stopping server lets the requests in
	// flight finish: 30s when 0.
	ShutdownTimeout time.Duration
}
