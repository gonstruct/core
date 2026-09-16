// Package otel wires OpenTelemetry for the application: an OTLP provider driven
// by configuration, and the tracing middleware that records each request.
//
// Bodies are attached to a span only when a request fails, and are masked
// first. Successful requests stay lean, which is what keeps always-on capture
// close to free.
//
// This package decides what to send. It does not sample — head sampling cannot
// know whether a request failed, because that is only known once the request
// finishes. Keeping a percentage plus every error is a collector's job.
package otel

import (
	"context"
	"github.com/gonstruct/core/otel/masking"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

type Otel struct {
	enabled     bool
	namespace   string
	name        string
	version     string
	environment string
	endpoint    string
	headers     map[string]string

	masker      *masking.Masker
	maxBodySize int
	captureBody bool
	attributeFn func(*gin.Context) []attribute.KeyValue

	engine *Engine
}

func New(options ...Option) *Otel {
	otel := &Otel{
		masker:      masking.New(),
		maxBodySize: DefaultMaxBodySize,
		captureBody: true,
	}

	for _, option := range options {
		option(otel)
	}

	return otel
}

// Enabled reports whether telemetry is switched on and has somewhere to go.
// When it is not, every other call is a no-op, so callers never need to branch.
func (self *Otel) Enabled() bool {
	return self.enabled && self.endpoint != ""
}

// Service is the value exported as service.name.
func (self *Otel) Service() string {
	if self.namespace == "" {
		return self.name
	}

	if self.name == "" {
		return self.namespace
	}

	return self.namespace + "-" + self.name
}

// Register installs the tracer and meter providers globally.
func (self *Otel) Register(context context.Context) error {
	if !self.Enabled() {
		return nil
	}

	engine, err := NewEngine(context, self)
	if err != nil {
		return err
	}

	self.engine = engine

	return nil
}

// Shutdown flushes pending telemetry and tears the providers down.
func (self *Otel) Shutdown(context context.Context) error {
	if self.engine == nil {
		return nil
	}

	return self.engine.Shutdown(context)
}
