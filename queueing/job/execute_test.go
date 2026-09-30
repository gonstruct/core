package job_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gonstruct/core/queueing/job"

	coreotel "github.com/gonstruct/core/otel"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type stubJob struct{ err error }

func (self *stubJob) Name() string                      { return "stub" }
func (self *stubJob) Handle(context.Context) error      { return self.err }
func (self *stubJob) Construct([]byte) (job.Job, error) { return self, nil }
func (self *stubJob) Failed(context.Context, error)     {}

func record(t *testing.T, jobErr error) sdktrace.ReadOnlySpan {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))

	coreotel.Bind(coreotel.New(
		coreotel.WithEnabled(true),
		coreotel.WithEndpoint("localhost:4318"),
		coreotel.WithService("example", "api"),
	))
	t.Cleanup(func() { coreotel.Bind(nil) })

	job.Execute(context.Background(), &stubJob{err: jobErr})

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	return spans[0]
}

// A rate limiter releasing a job is routine. Reporting it as an error buries
// the real failures.
func TestReleaseIsNotAnError(t *testing.T) {
	span := record(t, job.SilentRelease(5*time.Second))

	if span.Status().Code == codes.Error {
		t.Error("a released job must not be recorded as failed")
	}
}

func TestDontReleaseIsNotAnError(t *testing.T) {
	span := record(t, job.DontRelease("nothing to do"))

	if span.Status().Code == codes.Error {
		t.Error("a dontRelease directive must not be recorded as failed")
	}
}

func TestRealFailureIsAnError(t *testing.T) {
	span := record(t, errors.New("smtp refused the connection"))

	if span.Status().Code != codes.Error {
		t.Error("a genuine failure must still be recorded as an error")
	}
}

type hydratingJob struct {
	stubJob

	hydrated bool
}

func (self *hydratingJob) Hydrate(ctx context.Context) error {
	_, span := otel.Tracer("test").Start(ctx, "SELECT assets")
	span.End()
	self.hydrated = true

	return nil
}

// Hydration reads the database. Outside the job's span every query it makes
// is a trace of its own, with nothing to say which job it was for.
func TestHydrationRunsInsideTheJobSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	coreotel.Bind(coreotel.New(
		coreotel.WithEnabled(true),
		coreotel.WithEndpoint("localhost:4318"),
		coreotel.WithService("example", "api"),
	))
	t.Cleanup(func() { coreotel.Bind(nil) })

	current := &hydratingJob{}
	job.Execute(context.Background(), current)

	spans := recorder.Ended()
	if !current.hydrated || len(spans) != 2 {
		t.Fatalf("expected the job hydrated and two spans, got %d", len(spans))
	}
	if spans[0].Parent().SpanID() != spans[1].SpanContext().SpanID() {
		t.Error("the hydration query should be a child of the job span")
	}
}
