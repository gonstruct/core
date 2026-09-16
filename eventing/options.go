package eventing

type Option func(*Eventing)

func WithSubscribers(registrars ...func(*Engine)) Option {
	return func(eventing *Eventing) {
		for _, registrar := range registrars {
			registrar(eventing.engine)
		}
	}
}
