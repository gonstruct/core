package otel_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	coreotel "github.com/gonstruct/core/otel"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func backgroundSetup(t *testing.T) (*coreotel.Otel, *tracetest.SpanRecorder) {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))

	return coreotel.New(
		coreotel.WithEnabled(true),
		coreotel.WithEndpoint("localhost:4318"),
		coreotel.WithService("contentpad", "api"),
	), recorder
}

// A job that fails at 3am has no request behind it. It must still produce an
// error trace of its own, or the collector has nothing to keep.
func TestFailedJobRecordsAnErrorSpan(t *testing.T) {
	instance, recorder := backgroundSetup(t)

	_, finish := instance.StartJob(context.Background(), "send-welcome-email")
	finish(errors.New("smtp refused the connection"))

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if spans[0].Name() != "job send-welcome-email" {
		t.Errorf("unexpected span name %q", spans[0].Name())
	}

	if spans[0].Status().Code != codes.Error {
		t.Errorf("expected an error status, got %v", spans[0].Status().Code)
	}

	if len(spans[0].Events()) == 0 {
		t.Error("expected the error to be recorded on the span")
	}
}

func TestSucceedingJobIsNotAnError(t *testing.T) {
	instance, recorder := backgroundSetup(t)

	_, finish := instance.StartJob(context.Background(), "send-welcome-email")
	finish(nil)

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if spans[0].Status().Code == codes.Error {
		t.Error("a job that returned nil must not be marked as failed")
	}
}

func TestFailedCommandRecordsAnErrorSpan(t *testing.T) {
	instance, recorder := backgroundSetup(t)

	_, finish := instance.StartCommand(context.Background(), "prune-sessions")
	finish(errors.New("database unreachable"))

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	if spans[0].Name() != "command prune-sessions" {
		t.Errorf("unexpected span name %q", spans[0].Name())
	}

	if spans[0].Status().Code != codes.Error {
		t.Errorf("expected an error status, got %v", spans[0].Status().Code)
	}
}

// Disabled telemetry must not force callers to branch, and must not panic on
// the finish function.
func TestBackgroundSpansAreNoOpsWhenDisabled(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))

	instance := coreotel.New()

	_, finishJob := instance.StartJob(context.Background(), "send-welcome-email")
	finishJob(errors.New("boom"))

	_, finishCommand := instance.StartCommand(context.Background(), "prune-sessions")
	finishCommand(nil)

	if spans := recorder.Ended(); len(spans) != 0 {
		t.Errorf("expected no spans when disabled, got %d", len(spans))
	}
}

// A panicking job is the one run you most need to see, and it is the one that
// vanishes if finish is not deferred: finish calls span.End, and a span that
// never ends is never exported at all.
//
// This mirrors the guard the job runners wrap around Handle.
func TestPanickingJobStillEndsItsSpanAsAnError(t *testing.T) {
	instance, recorder := backgroundSetup(t)

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Error("expected the panic to be re-raised for the caller")
			}
		}()

		_, finish := instance.StartJob(context.Background(), "rotate-groups")

		defer func() {
			if recovered := recover(); recovered != nil {
				finish(fmt.Errorf("panic: %v", recovered))

				panic(recovered)
			}
		}()

		panic("nil map write")
	}()

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected the span to be ended despite the panic, got %d", len(spans))
	}

	if spans[0].Status().Code != codes.Error {
		t.Errorf("expected the span to be marked failed, got %v", spans[0].Status().Code)
	}
}
