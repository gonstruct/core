package seeder

import (
	"fmt"
	"github.com/gonstruct/core/console"
	"strings"
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
