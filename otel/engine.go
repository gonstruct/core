package otel

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

const metricInterval = 30 * time.Second

// Engine owns the SDK providers for the process.
type Engine struct {
	tracerProvider *trace.TracerProvider
	meterProvider  *metric.MeterProvider
	loggerProvider *sdklog.LoggerProvider
}

func NewEngine(context context.Context, config *Otel) (*Engine, error) {
	resource, err := newResource(context, config)
	if err != nil {
		return nil, err
	}

	traceExporter, err := otlptracehttp.New(context,
		otlptracehttp.WithEndpointURL(endpointURL(config.endpoint, "/v1/traces")),
		otlptracehttp.WithHeaders(config.headers),
	)
	if err != nil {
		return nil, fmt.Errorf("[otel] failed to create trace exporter: %w", err)
	}

	logExporter, err := otlploghttp.New(context,
		otlploghttp.WithEndpointURL(endpointURL(config.endpoint, "/v1/logs")),
		otlploghttp.WithHeaders(config.headers),
	)
	if err != nil {
		return nil, fmt.Errorf("[otel] failed to create log exporter: %w", err)
	}

	metricExporter, err := otlpmetrichttp.New(context,
		otlpmetrichttp.WithEndpointURL(endpointURL(config.endpoint, "/v1/metrics")),
		otlpmetrichttp.WithHeaders(config.headers),
	)
	if err != nil {
		return nil, fmt.Errorf("[otel] failed to create metric exporter: %w", err)
	}

	engine := &Engine{
		tracerProvider: trace.NewTracerProvider(
			trace.WithResource(resource),
			trace.WithBatcher(traceExporter),
		),
		meterProvider: metric.NewMeterProvider(
			metric.WithResource(resource),
			metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(metricInterval))),
		),
		loggerProvider: sdklog.NewLoggerProvider(
			sdklog.WithResource(resource),
			sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		),
	}

	otel.SetTracerProvider(engine.tracerProvider)
	otel.SetMeterProvider(engine.meterProvider)
	global.SetLoggerProvider(engine.loggerProvider)

	// TraceContext carries the trace across service boundaries; Baggage carries
	// arbitrary key/values alongside it. Without these the browser, the Next
	// server, and this API produce three unrelated traces per request.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return engine, nil
}

func (self *Engine) Shutdown(context context.Context) error {
	if err := self.tracerProvider.Shutdown(context); err != nil {
		return fmt.Errorf("[otel] failed to shutdown tracer provider: %w", err)
	}

	if err := self.meterProvider.Shutdown(context); err != nil {
		return fmt.Errorf("[otel] failed to shutdown meter provider: %w", err)
	}

	if err := self.loggerProvider.Shutdown(context); err != nil {
		return fmt.Errorf("[otel] failed to shutdown logger provider: %w", err)
	}

	return nil
}

func newResource(context context.Context, config *Otel) (*resource.Resource, error) {
	attributes := []attribute.KeyValue{
		semconv.ServiceName(config.Service()),
		semconv.ServiceInstanceID(os.Getenv("HOSTNAME")),
	}

	if config.namespace != "" {
		attributes = append(attributes, semconv.ServiceNamespace(config.namespace))
	}

	if config.environment != "" {
		attributes = append(attributes, semconv.DeploymentEnvironmentName(config.environment))
	}

	if config.version != "" {
		attributes = append(attributes,
			semconv.ServiceVersion(config.version),
			attribute.String(AttributeDeploymentCommitSHA, config.version),
		)
	}

	built, err := resource.New(context, resource.WithAttributes(attributes...))
	if err != nil {
		return nil, fmt.Errorf("[otel] failed to create resource: %w", err)
	}

	return built, nil
}

// endpointURL builds the signal URL from the configured endpoint.
//
// A bare host means the hosted backend, which is always TLS. An explicit
// scheme is honoured as given, so a collector on the internal network can be
// reached over plain http.
func endpointURL(endpoint string, path string) string {
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return strings.TrimSuffix(endpoint, "/") + path
	}

	return "https://" + strings.TrimSuffix(endpoint, "/") + path
}
