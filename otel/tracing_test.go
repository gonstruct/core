package otel_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	coreotel "github.com/gonstruct/core/otel"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setup(t *testing.T, options ...coreotel.Option) (*gin.Engine, *tracetest.SpanRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))

	options = append([]coreotel.Option{
		coreotel.WithEnabled(true),
		coreotel.WithEndpoint("localhost:4318"),
		coreotel.WithService("contentpad", "api"),
	}, options...)

	engine := gin.New()
	engine.Use(coreotel.New(options...).Tracing())

	return engine, recorder
}

func attributesOf(t *testing.T, recorder *tracetest.SpanRecorder) map[string]string {
	t.Helper()

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	found := map[string]string{}
	for _, attr := range spans[0].Attributes() {
		found[string(attr.Key)] = attr.Value.String()
	}

	return found
}

func TestServiceNameIsNamespaced(t *testing.T) {
	service := coreotel.New(coreotel.WithService("contentpad", "api")).Service()

	if service != "contentpad-api" {
		t.Errorf("expected contentpad-api, got %q", service)
	}
}

func TestDisabledWithoutEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	if coreotel.New(coreotel.WithEnabled(true)).Enabled() {
		t.Error("must not be enabled without an endpoint")
	}

	if coreotel.New().Enabled() {
		t.Error("must not be enabled unless switched on")
	}
}

func TestBodiesAreAttachedAndMaskedOnErrors(t *testing.T) {
	engine, recorder := setup(t)
	engine.POST("/users", func(context *gin.Context) {
		_, _ = context.Request.Body.Read(make([]byte, 1024))
		context.JSON(http.StatusUnprocessableEntity, gin.H{"error": "bad"})
	})

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users", strings.NewReader(`{"email":"a@b.com"}`))
	engine.ServeHTTP(httptest.NewRecorder(), request)

	found := attributesOf(t, recorder)

	if _, ok := found[coreotel.AttributeResponseBody]; !ok {
		t.Error("expected a response body on an error response")
	}

	if body := found[coreotel.AttributeRequestBody]; strings.Contains(body, "a@b.com") {
		t.Errorf("request body was not masked: %s", body)
	}
}

func TestBodiesAreOmittedOnSuccess(t *testing.T) {
	engine, recorder := setup(t)
	engine.POST("/users", func(context *gin.Context) {
		_, _ = context.Request.Body.Read(make([]byte, 1024))
		context.JSON(http.StatusOK, gin.H{"ok": true})
	})

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/users", strings.NewReader(`{"email":"a@b.com"}`))
	engine.ServeHTTP(httptest.NewRecorder(), request)

	found := attributesOf(t, recorder)

	if _, ok := found[coreotel.AttributeRequestBody]; ok {
		t.Error("request body must not be attached to a successful response")
	}

	if _, ok := found[coreotel.AttributeResponseBody]; ok {
		t.Error("response body must not be attached to a successful response")
	}
}

func TestApplicationAttributesReachTheSpan(t *testing.T) {
	engine, recorder := setup(t, coreotel.WithAttributes(func(*gin.Context) []attribute.KeyValue {
		return []attribute.KeyValue{attribute.String(coreotel.AttributeUserID, "user-1")}
	}))
	engine.GET("/me", func(context *gin.Context) { context.Status(http.StatusOK) })

	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/me", nil))

	if found := attributesOf(t, recorder); found[coreotel.AttributeUserID] != "user-1" {
		t.Errorf("application attribute missing: %v", found)
	}
}

func TestRequestIDIsRecorded(t *testing.T) {
	engine, recorder := setup(t)
	engine.GET("/me", func(context *gin.Context) { context.Status(http.StatusOK) })

	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/me", nil))

	if found := attributesOf(t, recorder); found[coreotel.AttributeRequestID] == "" {
		t.Error("request.id missing — spans cannot be correlated with logs")
	}
}

func TestHealthEndpointIsNotTraced(t *testing.T) {
	engine, recorder := setup(t)
	engine.GET("/health", func(context *gin.Context) { context.Status(http.StatusOK) })

	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))

	if spans := recorder.Ended(); len(spans) != 0 {
		t.Errorf("expected /health to be filtered out, got %d spans", len(spans))
	}
}

// Span names must use the route template, otherwise /users/1 and /users/2
// become separate operations and the backend sees thousands of them.
func TestSpanNameUsesRouteTemplate(t *testing.T) {
	engine, recorder := setup(t)
	engine.GET("/users/:id", func(context *gin.Context) { context.Status(http.StatusOK) })

	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/users/123", nil))

	spans := recorder.Ended()
	if len(spans) != 1 || spans[0].Name() != "GET /users/:id" {
		t.Errorf("unexpected span name: %v", spans)
	}
}
