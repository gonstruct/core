package otel_test

import (
	"context"
	"sync"
	"testing"

	coreotel "github.com/gonstruct/core/otel"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// recorder collects records in memory. The SDK ships no test processor at
// v0.21.0, and this is all the test needs.
type recorder struct {
	mutex   sync.Mutex
	records []sdklog.Record
}

func (self *recorder) Enabled(context.Context, sdklog.EnabledParameters) bool { return true }
func (self *recorder) Shutdown(context.Context) error                         { return nil }
func (self *recorder) ForceFlush(context.Context) error                       { return nil }

func (self *recorder) OnEmit(_ context.Context, record *sdklog.Record) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	self.records = append(self.records, *record)

	return nil
}

func loggerSetup(t *testing.T) (zerolog.Logger, *recorder) {
	t.Helper()

	recorder := new(recorder)
	global.SetLoggerProvider(sdklog.NewLoggerProvider(sdklog.WithProcessor(recorder)))

	coreotel.Bind(coreotel.New(
		coreotel.WithEnabled(true),
		coreotel.WithEndpoint("localhost:4318"),
		coreotel.WithService("example", "api"),
	))
	t.Cleanup(func() { coreotel.Bind(nil) })

	return zerolog.New(coreotel.NewLogWriter()).Hook(coreotel.TraceHook{}), recorder
}

func emitted(t *testing.T, source *recorder) []sdklog.Record {
	t.Helper()

	source.mutex.Lock()
	defer source.mutex.Unlock()

	return source.records
}

// The whole reason for a custom bridge: the official one drops every field,
// so an error arrives with no error on it.
func TestFieldsSurvive(t *testing.T) {
	logger, recorder := loggerSetup(t)

	logger.Warn().Str("limiter", "member-conversation").Int("attempt", 2).Msg("rate limiter failed")

	records := emitted(t, recorder)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	found := map[string]string{}
	records[0].WalkAttributes(func(pair attribute.KeyValue) bool {
		found[string(pair.Key)] = pair.Value.String()

		return true
	})

	if found["limiter"] != "member-conversation" {
		t.Errorf("expected limiter attribute, got %q", found["limiter"])
	}

	if found["attempt"] != "2" {
		t.Errorf("expected attempt attribute, got %q", found["attempt"])
	}
}

func TestBelowWarnIsNotShipped(t *testing.T) {
	logger, recorder := loggerSetup(t)

	logger.Info().Msg("routine")
	logger.Debug().Msg("noisy")

	if records := emitted(t, recorder); len(records) != 0 {
		t.Fatalf("expected nothing below warn, got %d records", len(records))
	}
}

func TestWarnAndAboveAreShipped(t *testing.T) {
	logger, recorder := loggerSetup(t)

	logger.Warn().Msg("degraded")
	logger.Error().Msg("broken")

	if records := emitted(t, recorder); len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
}

func TestNothingShipsWhenDisabled(t *testing.T) {
	recorder := new(recorder)
	global.SetLoggerProvider(sdklog.NewLoggerProvider(sdklog.WithProcessor(recorder)))

	coreotel.Bind(coreotel.New(coreotel.WithEnabled(false)))
	t.Cleanup(func() { coreotel.Bind(nil) })

	disabled := zerolog.New(coreotel.NewLogWriter())
	disabled.Warn().Msg("degraded")

	if records := emitted(t, recorder); len(records) != 0 {
		t.Fatalf("expected nothing to ship while disabled, got %d records", len(records))
	}
}

// A record logged with .Ctx must arrive carrying that span, or the backend
// cannot put the log next to the trace. The hook and the writer only manage it
// together, so this covers both halves.
func TestTraceContextIsCorrelated(t *testing.T) {
	logger, recorder := loggerSetup(t)

	provider := sdktrace.NewTracerProvider()
	traced, span := provider.Tracer("test").Start(context.Background(), "request")
	defer span.End()

	logger.Warn().Ctx(traced).Msg("failed inside a request")

	records := emitted(t, recorder)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	if records[0].TraceID() != span.SpanContext().TraceID() {
		t.Errorf("expected the record to carry the span's trace id, got %v", records[0].TraceID())
	}

	if records[0].SpanID() != span.SpanContext().SpanID() {
		t.Errorf("expected the record to carry the span id, got %v", records[0].SpanID())
	}
}

// The ids travel as fields so the writer can see them. They must not also be
// left on the record as attributes.
func TestTraceIdsAreNotDuplicatedAsAttributes(t *testing.T) {
	logger, recorder := loggerSetup(t)

	provider := sdktrace.NewTracerProvider()
	traced, span := provider.Tracer("test").Start(context.Background(), "request")
	defer span.End()

	logger.Warn().Ctx(traced).Msg("failed inside a request")

	records := emitted(t, recorder)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	records[0].WalkAttributes(func(pair attribute.KeyValue) bool {
		if pair.Key == "trace_id" || pair.Key == "span_id" {
			t.Errorf("%s should live on the record, not as an attribute", pair.Key)
		}

		return true
	})
}

// Without a context there is no span, and that is correct: the record happened
// outside a request. It must still ship.
func TestRecordsWithoutContextStillShip(t *testing.T) {
	logger, recorder := loggerSetup(t)

	logger.Warn().Msg("started up")

	records := emitted(t, recorder)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	if records[0].TraceID().IsValid() {
		t.Error("expected no trace id on a record logged without a context")
	}
}
