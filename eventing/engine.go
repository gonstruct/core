package eventing

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// A failing handler gets maxHandlerAttempts in total: the first inline, the
// rest in the background with exponential backoff. Handlers must therefore
// tolerate re-execution: do one atomic thing, or start with an "already
// done?" check.
const maxHandlerAttempts = 12

func NewEngine(options ...EngineOption) *Engine {
	engine := &Engine{
		handlers:   make(map[string][]handler),
		registered: make(map[string]string),
		retryDelay: defaultRetryDelay,
	}

	for _, option := range options {
		option(engine)
	}

	return engine
}

type Engine struct {
	handlers   map[string][]handler
	registered map[string]string
	retryDelay func(retry int) time.Duration
}

type EngineOption func(*Engine)

// WithRetryDelay overrides the backoff between handler retries. Tests use it
// to retry instantly.
func WithRetryDelay(delay func(retry int) time.Duration) EngineOption {
	return func(engine *Engine) {
		engine.retryDelay = delay
	}
}

// defaultRetryDelay doubles from 5s and caps at 10 minutes: 5s, 10s, 20s,
// 40s, 80s, ... — roughly 45 minutes of trying before a handler gives up.
func defaultRetryDelay(retry int) time.Duration {
	delay := 5 * time.Second << (retry - 1)
	if delay > 10*time.Minute {
		return 10 * time.Minute
	}

	return delay
}

type handler struct {
	name string
	call func(context.Context, Event) error
}

// run executes the handler and keeps it alive: the first attempt is inline,
// so a handler that works costs nothing extra, and the rest run in the
// background with backoff until the handler succeeds or the attempt budget
// runs out. The caller is never blocked and never told, because a failing
// handler must not break the thing that already happened.
//
// The retried context has its cancellation stripped: the request that
// dispatched the event is long gone, but its values (tracing) stay useful.
// Retries live in this process only — a restart loses them, the accepted cost
// of staying queue-free.
func (self handler) run(context context.Context, event Event, delay func(retry int) time.Duration) {
	err := self.call(context, event)
	if err == nil {
		return
	}

	log.Error().Err(err).
		Str("handler", self.name).
		Str("event", event.EventName()).
		Int("attempt", 1).
		Int("maxAttempts", maxHandlerAttempts).
		Msg("[eventing] handler failed")

	go self.retry(withoutCancel(context), event, delay)
}

func (self handler) retry(context context.Context, event Event, delay func(retry int) time.Duration) {
	for attempt := 2; attempt <= maxHandlerAttempts; attempt++ {
		time.Sleep(delay(attempt - 1))

		err := self.call(context, event)
		if err == nil {
			log.Info().
				Str("handler", self.name).
				Str("event", event.EventName()).
				Int("attempt", attempt).
				Msg("[eventing] handler recovered")

			return
		}

		log.Error().Err(err).
			Str("handler", self.name).
			Str("event", event.EventName()).
			Int("attempt", attempt).
			Int("maxAttempts", maxHandlerAttempts).
			Msg("[eventing] handler failed")
	}

	log.Error().
		Str("handler", self.name).
		Str("event", event.EventName()).
		Int("attempts", maxHandlerAttempts).
		Msg("[eventing] handler gave up, event lost for this handler")
}

// On starts a registration for one event type. Chain Subscribe to attach the
// subscribers:
//
//	events.On[WalletBalanceLowEvent](engine).Subscribe(
//		new(DispatchWebhookEventSubscriber),
//	)
//
// The registration file reads as a sentence: on this event, these subscribers
// react.
func On[E Event](engine *Engine) registration[E] {
	return registration[E]{engine: engine}
}

type registration[E Event] struct {
	engine *Engine
}

// Subscribe attaches subscribers to the event, at boot. The compiler enforces
// that a subscriber can handle the event, so a miswire never reaches runtime.
//
// Handlers are stored under the event's EventName, which is what dispatch
// looks up. Two events answering to one name would silently share handlers,
// so that panics here.
func (self registration[E]) Subscribe(subscribers ...Subscriber[E]) {
	var event E
	name := event.EventName()
	eventType := fmt.Sprintf("%T", event)

	if registered, taken := self.engine.registered[name]; taken && registered != eventType {
		panic(fmt.Sprintf(
			"[eventing] %s and %s both answer to %q: the name routes the event, so two events cannot share one",
			registered, eventType, name,
		))
	}

	self.engine.registered[name] = eventType

	for _, subscriber := range subscribers {
		bound := bind(name, eventType, subscriber)
		self.engine.handlers[name] = append(self.engine.handlers[name], bound)

		log.Info().Msgf("[eventing] %s -> %s", name, bound.name)
	}
}

// bind wraps a typed subscriber as an untyped handler. The type assertion
// only fails when an event answers to a name it did not register under, which
// is a wiring mistake rather than a runtime condition: it is reported once and
// never retried, because retrying cannot change the answer.
func bind[E Event](name string, eventType string, subscriber Subscriber[E]) handler {
	return handler{
		name: fmt.Sprintf("%T", subscriber),
		call: func(context context.Context, event Event) error {
			typed, ok := event.(E)
			if !ok {
				log.Error().
					Str("event", name).
					Str("dispatched", fmt.Sprintf("%T", event)).
					Msgf("[eventing] %s handles %s, so this dispatch is dropped", name, eventType)

				return nil
			}

			return subscriber.Handle(context, typed)
		},
	}
}

// DispatchEvent runs each event's handlers in registration order, events in
// the given order. Dispatching an event is a statement of fact: a failing
// handler never breaks the origin, the handlers after it, or the events after
// it. Each handler owns its own durability — see handler.run. An event
// without handlers is silent.
func (self *Engine) DispatchEvent(context context.Context, events ...Event) {
	for _, event := range events {
		if event == nil {
			continue
		}

		for _, handler := range self.handlers[event.EventName()] {
			handler.run(context, event, self.retryDelay)
		}
	}
}

func withoutCancel(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}
