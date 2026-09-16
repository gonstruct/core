package app

import (
	"context"
	"fmt"
	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/eventing"
	"github.com/gonstruct/core/facades/database"
	"github.com/gonstruct/core/queueing/job"
	"time"
)

func (self *App) DB() database.ContextExecutor {
	return self.db
}

func (self *App) Cache() *cache.Cache {
	if self.cache == nil {
		self.cache = cache.New()
	}

	return self.cache
}

func (self *App) Dispatch(ctx context.Context, job job.Job, delay ...time.Duration) error {
	if self.queueing == nil {
		return fmt.Errorf("cannot dispatch job: queueing is not initialized")
	}

	return self.queueing.Dispatch(ctx, job, delay...)
}

// DispatchEvent announces events to the subscribers registered for them in
// routes/subscribers.routes.go. Dispatching an event is a statement of fact:
// a failing handler is logged and retried by the engine and never breaks the
// thing that already happened.
func (self *App) DispatchEvent(context context.Context, events ...eventing.Event) {
	if self.eventing == nil {
		panic("eventing not configured: add app.WithEventing() to app.New()")
	}

	self.eventing.DispatchEvent(context, events...)
}

// Global accessors for the application instance bound by SetInstance().
// These are used by entrypoints and tests to access the application without
// passing it around. They panic if no application is bound.

func DB() database.ContextExecutor {
	return Instance().DB()
}

func Cache() *cache.Cache {
	return Instance().Cache()
}

func Dispatch(ctx context.Context, job job.Job, delay ...time.Duration) error {
	return Instance().Dispatch(ctx, job, delay...)
}

func DispatchEvent(context context.Context, events ...eventing.Event) {
	Instance().DispatchEvent(context, events...)
}
