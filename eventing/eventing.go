package eventing

import "context"

type Eventing struct {
	engine *Engine
}

// New builds the engine and runs the registrars immediately, so a subscriber
// missing a handler panics at boot instead of at the first dispatch.
func New(options ...Option) *Eventing {
	eventing := &Eventing{engine: NewEngine()}

	for _, option := range options {
		option(eventing)
	}

	return eventing
}

func (self *Eventing) DispatchEvent(context context.Context, events ...Event) {
	self.engine.DispatchEvent(context, events...)
}
