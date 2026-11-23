package scheduling

type Option func(*Scheduling)

// WithEnvironment is the environment commands limited to some run in. The
// app passes its own.
func WithEnvironment(environment string) Option {
	return func(r *Scheduling) {
		r.Environment = environment
	}
}

func WithSchedulers(schedulers ...func(*Engine)) Option {
	return func(r *Scheduling) {
		r.Schedulers = append(r.Schedulers, schedulers...)
	}
}
