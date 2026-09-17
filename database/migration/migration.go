package migration

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/gonstruct/core/config"
	"github.com/gonstruct/core/facades/database"

	goose "github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

type Migration struct {
	files       embed.FS
	environment config.Environment
	database    database.ContextExecutor
}

func New(files embed.FS, environment config.Environment, database database.ContextExecutor) *Migration {
	return &Migration{
		files:       files,
		environment: environment,
		database:    database,
	}
}

func (self *Migration) provider() (*goose.Provider, error) {
	connection, ok := self.database.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("migrations require a *sql.DB connection, got %T", self.database)
	}

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("failed to create session locker: %w", err)
	}

	return goose.NewProvider(goose.DialectPostgres, connection, self.files,
		goose.WithTableName("migrations"),
		goose.WithAllowOutofOrder(true),
		goose.WithSessionLocker(locker),
		goose.WithLogger(logger{}),
	)
}

func (self *Migration) sources(provider *goose.Provider, state goose.State) ([]*goose.Source, error) {
	statuses, err := provider.Status(context.Background())
	if err != nil {
		return nil, err
	}

	sources := []*goose.Source{}
	for _, status := range statuses {
		if status.State == state {
			sources = append(sources, status.Source)
		}
	}

	return sources, nil
}

func (self *Migration) guard(operation string) error {
	if self.environment == config.LocalEnvironment || self.environment == config.TestingEnvironment || self.environment == config.PreviewEnvironment {
		return nil
	}

	return fmt.Errorf("%s is only allowed in local or testing environments, current environment: %s", operation, self.environment)
}
