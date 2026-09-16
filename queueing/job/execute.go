package job

import (
	"context"
	"fmt"
	"github.com/gonstruct/core/otel"
)

// Execute runs the job through middleware and normalizes directive handling.
// shouldContinue=false indicates a DontRelease directive and no retry/failure.
func Execute(ctx context.Context, current Job) (shouldContinue bool, err error) {
	ctx, finish := otel.Instance().StartJob(ctx, current.Name())

	// A panic must still end the span, and end it as a failure. finish is what
	// calls span.End(), so without this the run that crashed is the one that
	// leaves no trace at all — the exact inverse of what you want. The panic
	// is re-raised so the caller's own recovery is unchanged.
	defer func() {
		if recovered := recover(); recovered != nil {
			finish(fmt.Errorf("panic: %v", recovered))

			panic(recovered)
		}
	}()

	err = ExecuteWithMiddlewares(ctx, current)
	if err == nil {
		finish(nil)

		return true, nil
	}

	// DontRelease is a directive, not a failure: the job chose to stop.
	if _, ok := asDontRelease(err); ok {
		finish(nil)

		return false, nil
	}

	// Release is also a directive: the job asked to run again later, usually
	// because a rate limiter said not yet. The error carries that instruction
	// to the driver, so it must still be returned — but it is not a failure
	// and must not be reported as one.
	if _, ok := AsRelease(err); ok {
		finish(nil)

		return true, err
	}

	finish(err)

	return true, err
}
