package app

import (
	"embed"
	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/console"
	"github.com/gonstruct/core/database/migration"
	"github.com/gonstruct/core/eventing"
	"github.com/gonstruct/core/facades/database"
	"github.com/gonstruct/core/facades/database/seeder"
	"github.com/gonstruct/core/queueing"
	"github.com/gonstruct/core/routing"
	"github.com/gonstruct/core/scheduling"
)

type option func(*App)

func WithDatabase(db database.ContextExecutor) option {
	return func(a *App) {
		a.db = db
	}
}

func WithProviders(providers providers) option {
	return func(a *App) {
		if len(a.providers) == 0 {
			a.providers = []provider{}
		}

		a.providers = append(a.providers, providers...)
	}
}

func WithMigrations(files embed.FS) option {
	return func(a *App) {
		migrations := migration.New(files, a.environment, a.db)
		a.console.Add(migrations.Commands()...)
	}
}

func WithSeeders(seeders seeder.Seeders) option {
	return func(a *App) {
		a.console.Add(seeder.New(a.db, seeders).Command())
	}
}

func WithCache(options ...cache.Option) option {
	return func(a *App) {
		a.cache = cache.New(options...)
	}
}

func WithRouting(options ...routing.Option) option {
	return func(a *App) {
		a.routing = routing.New(options...)
		a.console.Register("serve", func(*console.Console, []string) error { return a.routing.Run() })
	}
}

func WithScheduling(options ...scheduling.Option) option {
	return func(a *App) {
		a.scheduling = scheduling.New(options...)
		a.console.Register("schedule:work", func(*console.Console, []string) error { return a.scheduling.Run() })
	}
}

func WithQueueing(options ...queueing.Option) option {
	return func(a *App) {
		a.queueing = queueing.New(options...)
		a.console.Register("queue:work", func(*console.Console, []string) error { return a.queueing.Run() })
	}
}

func WithEventing(options ...eventing.Option) option {
	return func(a *App) {
		a.eventing = eventing.New(options...)
	}
}
