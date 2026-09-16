package otel

import (
	"github.com/gonstruct/core/routing/response"
	"sync"

	"github.com/gin-gonic/gin"
)

// TracingMiddleware records a server span for each request. Register it in the
// global chain like any other middleware:
//
//	routing.WithMiddleware("global", new(otel.TracingMiddleware), ...)
//
// It is a no-op when telemetry is disabled, so it can be registered
// unconditionally.
type TracingMiddleware struct {
	once    sync.Once
	handler gin.HandlerFunc
}

// Handle resolves the instance on the first request, not at registration.
// Middleware is registered while app.New is being composed, and the provider
// that binds telemetry may not have run yet — resolving earlier would capture
// the disabled fallback and silently trace nothing.
//
// The handler is built once: otelgin resolves a tracer from the provider when
// its handler is constructed, so rebuilding it per request would repeat that
// work on every call.
func (self *TracingMiddleware) Handle(context *gin.Context) response.Response {
	self.once.Do(func() {
		self.handler = Instance().Tracing()
	})

	self.handler(context)

	return response.Next()
}
