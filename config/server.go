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
	// HealthEndpoint is the path that answers healthy while the server
	// runs, for a load balancer or a container's health check: none when
	// empty.
	HealthEndpoint string
	// TrustedPlatform is the header the platform in front of the server
	// puts the client's IP in, as gin's TrustedPlatform: none when empty.
	TrustedPlatform string
}
