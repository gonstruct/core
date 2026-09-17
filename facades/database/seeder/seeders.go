package seeder

import (
	"fmt"
	"strings"

	"github.com/gonstruct/core/console"
)

type seeder interface {
	Run()
}

type Seeders []seeder

func (self Seeders) Run() {
	for _, seed := range self {
		name := strings.TrimPrefix(fmt.Sprintf("%T", seed), "*")

		_ = console.Task(name, func() error {
			seed.Run()
			return nil
		})
	}
}
