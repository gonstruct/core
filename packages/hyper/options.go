package hyper

import (
	"context"
	"time"
)

func WithContext(ctx context.Context) requestOption {
	return func(request *Request) {
		request.ctx = ctx
	}
}

func WithBase(base string) requestOption {
	return func(request *Request) {
		request.base = base
	}
}

func WithTimeout(timeout time.Duration) requestOption {
	return func(request *Request) {
		request.timeout = timeout
	}
}
