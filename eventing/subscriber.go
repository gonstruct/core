package eventing

import "context"

// Subscriber reacts to one event. A subscriber for WalletBalanceLowEvent is
// any type with
//
//	func (S) Handle(context.Context, WalletBalanceLowEvent) error
//
// One subscriber handles one event: Go has no method overloading, so a type
// that must react to two events is two types. Returning an error asks the
// engine to retry — see maxHandlerAttempts.
type Subscriber[E Event] interface {
	Handle(context context.Context, event E) error
}
