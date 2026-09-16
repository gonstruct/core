package migration

import (
	"github.com/gonstruct/core/console"
)

func (self *Migration) Fresh() error {
	if err := self.guard("database fresh"); err != nil {
		return err
	}

	if err := console.Task("Dropping all tables", func() error {
		_, err := self.database.Exec("DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;")
		return err
	}); err != nil {
		return err
	}

	return self.Migrate(false)
}
