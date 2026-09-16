package otel

// The otel provider builds the instance and binds it here.
//
// It is deliberately not part of App: the application's composition —
// database, routing, queueing, scheduling, eventing — is what App is for.
// Telemetry is an extra a provider adds, and providers run before app.New,
// so it could not live on the instance anyway.

var instance *Otel

// Bind stores the instance built by the otel provider.
func Bind(bound *Otel) {
	instance = bound
}

// Instance is never nil. Without a binding it returns a disabled instance, so
// every call is a no-op and callers never branch.
func Instance() *Otel {
	if instance == nil {
		instance = New()
	}

	return instance
}
