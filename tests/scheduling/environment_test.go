package scheduling_test

import (
	"context"
	"testing"

	"github.com/gonstruct/core/scheduling"
	"github.com/gonstruct/core/scheduling/command"
)

type nightly struct{}

func (nightly) Name() string                 { return "nightly" }
func (nightly) Handle(context.Context) error { return nil }

// A command limited to some environments is left out of the others, as the
// app's config says, whatever APP_ENV is.
func TestCommandsRunInTheirEnvironmentsOnly(t *testing.T) {
	t.Setenv("APP_ENV", "production")

	for _, each := range []struct {
		environment string
		registered  bool
	}{{"production", true}, {"staging", false}} {
		engine, err := scheduling.NewEngine()
		if err != nil {
			t.Fatal(err)
		}
		engine.Environment = each.environment
		engine.Command(nightly{}, command.EveryMinute(), command.Environment[string]("production"))

		if got := len(engine.Jobs()) > 0; got != each.registered {
			t.Errorf("in %s: expected registered %v, got %v", each.environment, each.registered, got)
		}
	}
}
