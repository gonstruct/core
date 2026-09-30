package otel_test

import (
	"context"
	"testing"

	coreotel "github.com/gonstruct/core/otel"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// A job is its own trace, found again from the request that dispatched it
// through a link rather than by growing that request's trace.
func TestJobLinksToTheSpanThatDispatchedIt(t *testing.T) {
	instance, recorder := backgroundSetup(t)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	requestContext, request := otel.Tracer("test").Start(context.Background(), "POST /batches")
	carrier := coreotel.Carrier(requestContext)
	request.End()

	_, finish := instance.StartJob(coreotel.Dispatched(context.Background(), carrier), "runs.advance")
	finish(nil)

	job := recorder.Ended()[1]
	if job.Parent().IsValid() {
		t.Error("a job must start a trace of its own")
	}
	if len(job.Links()) != 1 || job.Links()[0].SpanContext.SpanID() != request.SpanContext().SpanID() {
		t.Errorf("expected a link to the dispatching span, got %v", job.Links())
	}
}

func TestJobRunInsideASpanIsStillARoot(t *testing.T) {
	instance, recorder := backgroundSetup(t)

	outer, span := otel.Tracer("test").Start(context.Background(), "worker")
	_, finish := instance.StartJob(outer, "runs.advance")
	finish(nil)
	span.End()

	if job := recorder.Ended()[0]; job.Parent().IsValid() {
		t.Error("a job must not become a child of whatever runs it")
	}
}

func TestNothingTracedDispatchesWithoutALink(t *testing.T) {
	instance, recorder := backgroundSetup(t)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	carrier := coreotel.Carrier(context.Background())
	_, finish := instance.StartJob(coreotel.Dispatched(context.Background(), carrier), "runs.advance")
	finish(nil)

	if links := recorder.Ended()[0].Links(); len(links) != 0 {
		t.Errorf("expected no link, got %v", links)
	}
}
