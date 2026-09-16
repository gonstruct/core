package job

import "context"

type Middleware interface {
	Handle(ctx context.Context, current Job, next func(context.Context) error) error
}

func ExecuteWithMiddlewares(ctx context.Context, current Job) error {
	execute := func(runCtx context.Context) error {
		return current.Handle(runCtx)
	}

	withMiddleware, ok := current.(JobWithMiddleware)
	if !ok {
		return execute(ctx)
	}

	middlewares := withMiddleware.Middlewares()
	for idx := len(middlewares) - 1; idx >= 0; idx-- {
		next := execute
		middleware := middlewares[idx]
		execute = func(runCtx context.Context) error {
			return middleware.Handle(runCtx, current, next)
		}
	}

	return execute(ctx)
}
