package otel

import (
	"github.com/gonstruct/core/otel/masking"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// DefaultMaxBodySize is how many bytes of each body are captured.
const DefaultMaxBodySize = 64 * 1024

type Option func(*Otel)

// WithService sets the identity of this service.
//
// service.name is exported as "<namespace>-<name>". Most backends group
// primarily by service.name and treat service.namespace as secondary, so a bare
// "api" would collide with every other project reporting to the same account.
// The namespace is sent separately as well, for backends that use it.
func WithService(namespace, name string) Option {
	return func(self *Otel) {
		self.namespace = namespace
		self.name = name
	}
}

// WithEnabled switches telemetry on. Off by default, so a service that never
// configures it produces no exporters and no network traffic.
func WithEnabled(enabled bool) Option {
	return func(self *Otel) { self.enabled = enabled }
}

// WithVersion records which build produced a span. Pass the commit SHA so
// "which deploy broke this" is answerable.
func WithVersion(version string) Option {
	return func(self *Otel) { self.version = version }
}

// WithEnvironment sets deployment.environment.name.
func WithEnvironment(environment string) Option {
	return func(self *Otel) { self.environment = environment }
}

// WithEndpoint sets the OTLP host. No scheme — "ingest.example.com".
func WithEndpoint(endpoint string) Option {
	return func(self *Otel) { self.endpoint = endpoint }
}

// WithHeaders sets the headers sent to the OTLP endpoint, usually auth.
func WithHeaders(headers map[string]string) Option {
	return func(self *Otel) { self.headers = headers }
}

// WithMasker replaces the default masker. Use masking.New("internal_ref") to
// keep the built-in keys and add your own.
func WithMasker(masker *masking.Masker) Option {
	return func(self *Otel) { self.masker = masker }
}

// WithMaxBodySize caps how much of each body is captured.
func WithMaxBodySize(size int) Option {
	return func(self *Otel) { self.maxBodySize = size }
}

// WithoutBodyCapture disables body capture entirely. Bodies are only ever
// attached to failed responses, but some services should not buffer them at all.
func WithoutBodyCapture() Option {
	return func(self *Otel) { self.captureBody = false }
}

// WithAttributes adds attributes to every server span. This is the seam for
// application context that core cannot know about — the authenticated user, a
// tenant, a feature flag.
//
//	otel.WithAttributes(func(context *gin.Context) []attribute.KeyValue {
//	    session, ok := providers.GetSessionSafe(context)
//	    if !ok {
//	        return nil
//	    }
//
//	    return []attribute.KeyValue{
//	        attribute.String(otel.AttributeUserID, session.UserID),
//	    }
//	})
func WithAttributes(fn func(*gin.Context) []attribute.KeyValue) Option {
	return func(self *Otel) { self.attributeFn = fn }
}
