package scheduling

type Option func(*Scheduling)

func WithSchedulers(schedulers ...func(*Engine)) Option {
	return func(r *Scheduling) {
		r.Schedulers = append(r.Schedulers, schedulers...)
	}
}
