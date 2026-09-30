package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type dispatcherKey struct{}

// Carrier is the trace context of whatever is dispatching a job, in a form a
// queue can store next to the payload. It is empty when nothing is traced.
func Carrier(ctx context.Context) map[string]string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	return carrier
}

// Dispatched puts the span that dispatched a job, read back from its carrier,
// on the context the job runs in, for StartJob to link to.
//
// It is a link and not a parent on purpose. A job can run minutes after the
// request that dispatched it, again on every release, and dispatch the next
// job in turn; as children, one request would grow a trace for as long as the
// work goes on, and a collector deciding per trace would keep it in pieces.
func Dispatched(ctx context.Context, carrier map[string]string) context.Context {
	dispatcher := trace.SpanContextFromContext(
		otel.GetTextMapPropagator().Extract(context.Background(), propagation.MapCarrier(carrier)),
	)
	if !dispatcher.IsValid() {
		return ctx
	}

	return context.WithValue(ctx, dispatcherKey{}, dispatcher)
}

func dispatcherLinks(ctx context.Context) []trace.Link {
	dispatcher, ok := ctx.Value(dispatcherKey{}).(trace.SpanContext)
	if !ok {
		return nil
	}

	return []trace.Link{{SpanContext: dispatcher}}
}
