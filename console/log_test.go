package console_test

import (
	"context"
	"github.com/gonstruct/core/console"
	"github.com/gonstruct/core/otel"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

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

// Framework output goes through the same logger as everything else, so a
// failed task reaches the collector. A successful one is Info and stays on the
// console, like every other record below warn.
func TestFailedTaskReachesTheCollector(t *testing.T) {
	recorder := new(recorder)
	global.SetLoggerProvider(sdklog.NewLoggerProvider(sdklog.WithProcessor(recorder)))

	otel.Bind(otel.New(
		otel.WithEnabled(true),
		otel.WithEndpoint("localhost:4318"),
		otel.WithService("example", "api"),
	))
	t.Cleanup(func() { otel.Bind(nil) })

	previous := log.Logger
	log.Logger = zerolog.New(otel.NewLogWriter()).Hook(otel.TraceHook{})
	t.Cleanup(func() { log.Logger = previous })

	provider := sdktrace.NewTracerProvider()
	traced, span := provider.Tracer("test").Start(context.Background(), "request")
	defer span.End()

	console.Request(traced, "GET", "/creators", 200, time.Millisecond, "req-1")

	if err := console.Task("seeding", func() error { return context.Canceled }); err == nil {
		t.Fatal("expected Task to return the error it was given")
	}

	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()

	if len(recorder.records) != 1 {
		t.Fatalf("expected only the failed task to ship, got %d records", len(recorder.records))
	}

	if recorder.records[0].Severity() != otellog.SeverityError {
		t.Errorf("expected the failed task to arrive as an error, got %v", recorder.records[0].Severity())
	}
}
