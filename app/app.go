package app

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/config"
	"github.com/gonstruct/core/console"
	"github.com/gonstruct/core/eventing"
	"github.com/gonstruct/core/facades/database"
	"github.com/gonstruct/core/queueing"
	"github.com/gonstruct/core/routing"
	"github.com/gonstruct/core/scheduling"

	"github.com/rs/zerolog/log"

	"github.com/gonstruct/validation/env"
)

type App struct {
	environment config.Environment

	ctx context.Context
	db  database.ContextExecutor

	createdAt time.Time
	startedAt time.Time

	providers providers
	console   *console.Console

	routing    *routing.Routing
	scheduling *scheduling.Scheduling
	queueing   *queueing.Queueing
	eventing   *eventing.Eventing
	cache      *cache.Cache

	// bindings is the container: what WithSingletons declared and Provide
	// added, keyed by type.
	bindings sync.Map
}

func Instance() *App {
	if instance == nil {
		panic("no app instance bound: call app.SetInstance(app.New(...)) first")
	}

	return instance
}

var instance *App

func New(options ...option) *App {
	application := &App{
		environment: env.Enum[config.Environment]("APP_ENV"),
		ctx:         context.Background(),
		createdAt:   time.Now(),
		console:     console.New(),
	}

	for _, option := range options {
		option(application)
	}

	if err := application.providers.Register(application.ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to register providers")
	}

	if err := application.providers.Boot(application.ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to boot providers")
	}

	return application
}

// SetInstance binds an application to the package level accessors such as
// app.DB() and app.Dispatch(). Entrypoints bind the application they boot.
// Tests can build applications without binding them.
func (self *App) SetInstance() *App {
	instance = self
	return self
}

func (self *App) Start() {
	self.SetInstance()

	arguments := os.Args[1:]
	if len(arguments) == 0 {
		log.Fatal().Msg("No command specified.")
	}

	self.startedAt = time.Now()

	if err := self.console.Run(arguments[0], arguments[1:]); err != nil {
		log.Fatal().Err(err).Msgf("Command failed: %s", arguments[0])
	}
}

func (self *App) Shutdown() {
	if closer, ok := self.db.(database.DatabaseCloser); ok {
		if err := closer.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close database connection")
		}
	}

	self.providers.Shutdown(self.ctx)
}
