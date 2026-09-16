package eventing

// Event is anything that can be dispatched.
//
// EventName is the stable identity of the fact, and the key handlers are
// registered under. The Go type name would work as a key too, but it changes
// the moment the struct is renamed. This one does not, which is what logs,
// and later broadcasts and webhooks, need.
type Event interface {
	EventName() string
}
