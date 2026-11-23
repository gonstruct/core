package config

import "time"

type Server struct {
	Host            string        // default 0.0.0.0
	Port            int           // default 80
	ShutdownTimeout time.Duration // default 30s
	HealthEndpoint  string        // no health endpoint when empty
	TrustedPlatform string        // see gin.Engine.TrustedPlatform
	ShowErrors      bool          // include the error in failed responses; never in production
}
