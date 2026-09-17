package migration

import (
	"context"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gonstruct/core/console"

	"github.com/rs/zerolog/log"

	goose "github.com/pressly/goose/v3"
)

func (self *Migration) Reset() error {
	if err := self.guard("database reset"); err != nil {
		return err
	}

	provider, err := self.provider()
	if err != nil {
		return err
	}

	applied, err := self.sources(provider, goose.StateApplied)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		log.Info().Msg("Database is empty, no migrations to roll back.")
		return nil
	}

	console.Section("Rolling back migrations.")

	for _, source := range slices.Backward(applied) {
		if err := console.Task(strings.TrimSuffix(filepath.Base(source.Path), ".sql"), func() error {
			_, err := provider.Down(context.Background())
			return err
		}); err != nil {
			return err
		}
	}

	return nil
}
