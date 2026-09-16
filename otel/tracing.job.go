package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartJob records a span for one job run.
//
// Jobs are not requests: nothing upstream started a trace, so this is a root
// span. A failed job therefore produces an error trace of its own, which the
// collector keeps under the same rule that keeps failed requests.
//
// The returned function must be called with the job's error, or nil. It is a
// no-op when telemetry is disabled, so callers never branch.
func (self *Otel) StartJob(ctx context.Context, name string) (context.Context, func(error)) {
	if !self.Enabled() {
		return ctx, func(error) {}
	}

	ctx, span := otel.Tracer(self.Service()).Start(ctx, "job "+name,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attribute.String(AttributeJobName, name)),
	)

	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
	}
}

// Start records a span for one unit of work inside a request — a call into a
// library that does not instrument itself, where the time would otherwise be
// invisible inside the parent.
//
// Unlike StartJob this continues the caller's trace rather than beginning one:
// the work belongs to whatever request asked for it.
//
// The returned function must be called with the work's error, or nil. It is a
// no-op when telemetry is disabled, so callers never branch.
func (self *Otel) Start(ctx context.Context, name string) (context.Context, func(error)) {
	if !self.Enabled() {
		return ctx, func(error) {}
	}

	ctx, span := otel.Tracer(self.Service()).Start(ctx, name, trace.WithSpanKind(trace.SpanKindClient))

	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
	}
}

// StartCommand records a span for one command run, scheduled or manual.
//
// Same shape as StartJob: a root span, so a failing scheduled command is
// visible on its own rather than only in a container log.
func (self *Otel) StartCommand(ctx context.Context, name string) (context.Context, func(error)) {
	if !self.Enabled() {
		return ctx, func(error) {}
	}

	ctx, span := otel.Tracer(self.Service()).Start(ctx, "command "+name,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attribute.String(AttributeCommandName, name)),
	)

	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}

		span.End()
	}
}
