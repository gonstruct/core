package migration

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/gonstruct/core/console"

	"github.com/rs/zerolog/log"

	goose "github.com/pressly/goose/v3"
)

func (self *Migration) Migrate(force bool) error {
	// The guard decides which environments may migrate unforced; --force is
	// the operator's override everywhere else.
	if !force {
		if err := self.guard("database migration without the --force flag"); err != nil {
			return err
		}
	}

	provider, err := self.provider()
	if err != nil {
		return err
	}

	pending, err := self.sources(provider, goose.StatePending)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		log.Info().Msg("Database is up to date, no migrations to run.")
		return nil
	}

	console.Section("Running migrations.")

	for _, source := range pending {
		if err := console.Task(strings.TrimSuffix(filepath.Base(source.Path), ".sql"), func() error {
			_, err := provider.UpByOne(context.Background())
			return err
		}); err != nil {
			return err
		}
	}

	return nil
}
