package seeder

import "github.com/gonstruct/core/console"

func (self *Seeder) Command() console.Command {
	return console.Command{
		Name: "db:seed",
		Handle: func(*console.Console, []string) error {
			self.Call()
			return nil
		},
	}
}
