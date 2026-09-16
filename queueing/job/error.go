package job

import (
	"errors"
	"fmt"
	"time"
)

func Fail(err error) error {
	if err == nil {
		err = fmt.Errorf("job failed without error")
	}

	return &failError{Cause: err}
}

type failError struct {
	Cause error
}

func (e *failError) Error() string {
	if e.Cause == nil {
		return "queue directive fail"
	}
	return e.Cause.Error()
}

func (e *failError) Unwrap() error {
	return e.Cause
}

func AsFail(err error) (*failError, bool) {
	var failErr *failError
	if errors.As(err, &failErr) {
		return failErr, true
	}
	return nil, false
}

func DontRelease(reasons ...string) error {
	reason := func() *string {
		if len(reasons) == 0 {
			return nil
		}
		return &reasons[0]
	}()

	return &dontReleaseError{Reason: reason}
}

type dontReleaseError struct {
	Reason *string
}

func (e *dontReleaseError) Error() string {
	if e.Reason == nil {
		return "queue directive dontRelease"
	}
	return fmt.Sprintf("queue directive dontRelease: %s", *e.Reason)
}

func asDontRelease(err error) (*dontReleaseError, bool) {
	var dontReleaseErr *dontReleaseError
	if errors.As(err, &dontReleaseErr) {
		return dontReleaseErr, true
	}
	return nil, false
}

func Release(delay time.Duration) error {
	if delay < 0 {
		delay = 0
	}

	return &releaseError{Delay: delay}
}

func SilentRelease(delay time.Duration) error {
	if delay < 0 {
		delay = 0
	}

	return &releaseError{Delay: delay, Silent: true}
}

type releaseError struct {
	Delay  time.Duration
	Silent bool
}

func (e *releaseError) Error() string {
	if e.Silent {
		return fmt.Sprintf("queue middleware silent release requested (delay=%s)", e.Delay)
	}

	return fmt.Sprintf("queue middleware release requested (delay=%s)", e.Delay)
}

func AsRelease(err error) (*releaseError, bool) {
	var releaseErr *releaseError
	if errors.As(err, &releaseErr) {
		return releaseErr, true
	}
	return nil, false
}
