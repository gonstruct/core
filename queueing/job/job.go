package job

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Queueable[T Job] struct{}

func (q *Queueable[T]) Construct(payload []byte) (Job, error) {
	var job T
	if err := json.Unmarshal(payload, &job); err != nil {
		return nil, err
	}
	return job, nil
}

// Fail marks the job as permanently failed. It will not be retried regardless
// of remaining attempts. Use this for unrecoverable errors.
func (q *Queueable[T]) Fail(err error) error {
	return Fail(err)
}

// Error returns a plain error that causes the job to be retried (up to its
// max attempts). Use this for transient/recoverable failures.
func (q *Queueable[T]) Error(format string, a ...any) error {
	return fmt.Errorf(format, a...)
}

// Release puts the job back on the queue after the given delay. The job will
// be picked up again once the delay has elapsed. This counts as a retry attempt
// and a failure. Use SilentRelease to avoid counting as a failure.
func (*Queueable[T]) Release(delay time.Duration) error {
	return Release(delay)
}

// SilentRelease re-queues the job after the given delay without counting
// as a failure or retry attempt.
func (*Queueable[T]) SilentRelease(reason ...string) error {
	return DontRelease(reason...)
}

// DontRelease silently discards the job. It will not be retried and is not
// counted as a failure. Use this when the work is no longer needed.
func (q *Queueable[T]) DontRelease(reason ...string) error {
	return DontRelease(reason...)
}

// Skip silently discards the job without retrying or marking it as failed.
// Alias for DontRelease.
func (q *Queueable[T]) Skip(reason ...string) error {
	return q.DontRelease(reason...)
}

// Skipf silently discards the job with a formatted reason string.
func (q *Queueable[T]) Skipf(format string, a ...any) error {
	return q.Skip(fmt.Sprintf(format, a...))
}

type Job interface {
	Construct([]byte) (Job, error)

	Name() string
	Handle(context.Context) error
	Failed(ctx context.Context, err error)
}

type JobWithDispatchPreparation interface {
	Job

	PrepareDispatch(ctx context.Context) (Job, error)
}

type JobWithMiddleware interface {
	Job
	Middlewares() []Middleware
}

type JobWithHydration interface {
	Job
	Hydrate(ctx context.Context) error
}

type JobWithFailure interface {
	Job

	Failure(ctx context.Context, err error)
}

type JobWithBackoff interface {
	Job
	Backoff(tries int) time.Duration
}

type JobWithTries interface {
	Job

	Tries() int
}

type JobWithQueue interface {
	Job

	Queue() string
}

type JobWithPriority interface {
	Job

	Priority() int
}

type JobWithoutOverlapping interface {
	Job

	Unique() string
}

type JobWithRetention interface {
	Job

	Retention() time.Duration
}
