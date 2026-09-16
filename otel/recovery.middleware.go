package otel

import (
	"fmt"
	"github.com/gonstruct/core/routing/response"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecoveryMiddleware records a panic on the span before letting it continue.
//
// gin.Recovery runs outside the tracing middleware, so by the time it turns a
// panic into a 500 the span has already ended — recording whatever the writer
// said at the time, which is 200. The trace then claims success for a request
// the caller watched fail, and there is nothing anywhere to say otherwise: the
// stack trace goes to stderr, which is not shipped.
//
// Register it directly after TracingMiddleware so it sits inside the span. It
// records the panic and re-panics, so gin still writes the response exactly as
// it did before.
type RecoveryMiddleware struct{}

func (middleware *RecoveryMiddleware) Handle(context *gin.Context) response.Response {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		span := trace.SpanFromContext(context.Request.Context())
		if span.IsRecording() {
			err, isError := recovered.(error)
			if !isError {
				err = fmt.Errorf("%v", recovered)
			}

			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, err.Error())
		}

		panic(recovered)
	}()

	context.Next()

	return response.Next()
}
