package migration

import (
	"flag"

	"github.com/gonstruct/core/console"
)

func (self *Migration) Commands() []console.Command {
	return []console.Command{
		{Name: "migrate", Handle: self.migrateCommand},
		{Name: "migrate:fresh", Handle: self.freshCommand},
		{Name: "migrate:reset", Handle: func(*console.Console, []string) error { return self.Reset() }},
		{Name: "migrate:refresh", Handle: func(*console.Console, []string) error { return self.Refresh() }},
	}
}

func (self *Migration) migrateCommand(_ *console.Console, arguments []string) error {
	flags := flag.NewFlagSet("migrate", flag.ExitOnError)
	force := flags.Bool("force", false, "Force migrations outside local and testing environments")
	if err := flags.Parse(arguments); err != nil {
		return err
	}

	return self.Migrate(*force)
}

func (self *Migration) freshCommand(console *console.Console, arguments []string) error {
	flags := flag.NewFlagSet("migrate:fresh", flag.ExitOnError)
	seed := flags.Bool("seed", false, "Seed the database after migrating")
	if err := flags.Parse(arguments); err != nil {
		return err
	}

	if err := self.Fresh(); err != nil {
		return err
	}

	if *seed {
		return console.Run("db:seed", nil)
	}

	return nil
}
