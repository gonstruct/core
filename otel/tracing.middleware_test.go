package otel_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	coreotel "github.com/gonstruct/core/otel"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func middlewareSetup(t *testing.T) (*gin.Engine, *tracetest.SpanRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))

	coreotel.Bind(nil)
	t.Cleanup(func() { coreotel.Bind(nil) })

	middleware := new(coreotel.TracingMiddleware)

	engine := gin.New()
	engine.Use(func(context *gin.Context) { middleware.Handle(context) })

	return engine, recorder
}

// The middleware is registered while app.New is being composed, before the
// provider binds telemetry. Resolving at registration would capture the
// disabled fallback and trace nothing, without any error to show for it.
func TestMiddlewareResolvesAfterRegistration(t *testing.T) {
	engine, recorder := middlewareSetup(t)

	engine.GET("/users/:id", func(context *gin.Context) { context.Status(http.StatusOK) })

	// The provider binds only now, after the middleware was registered.
	coreotel.Bind(coreotel.New(
		coreotel.WithEnabled(true),
		coreotel.WithEndpoint("localhost:4318"),
		coreotel.WithService("contentpad", "api"),
	))

	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/users/1", nil))

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected the late-bound instance to be traced, got %d spans", len(spans))
	}

	if spans[0].Name() != "GET /users/:id" {
		t.Errorf("expected GET /users/:id, got %q", spans[0].Name())
	}
}

func TestMiddlewareIsANoOpWhenDisabled(t *testing.T) {
	engine, recorder := middlewareSetup(t)

	served := false
	engine.GET("/users/:id", func(context *gin.Context) {
		served = true
		context.Status(http.StatusOK)
	})

	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/users/1", nil))

	if !served {
		t.Error("the chain must continue when telemetry is disabled")
	}

	if spans := recorder.Ended(); len(spans) != 0 {
		t.Errorf("expected no spans, got %d", len(spans))
	}
}
