package seeder

import (
	"github.com/gonstruct/core/console"
	"github.com/gonstruct/core/facades/database"

	"github.com/aarondl/sqlboiler/v4/boil"
)

func New(database database.ContextExecutor, seeders Seeders) *Seeder {
	return &Seeder{
		database: database,
		seeders:  seeders,
	}
}

type Seeder struct {
	database database.ContextExecutor
	seeders  Seeders
}

func (self *Seeder) Call() {
	boil.SetDB(self.database)
	defer boil.SetDB(nil)

	console.Section("Seeding database.")
	self.seeders.Run()
}
