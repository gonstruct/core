package scheduling

type Scheduling struct {
	Schedulers []func(*Engine)
}

func New(options ...Option) *Scheduling {
	scheduling := new(Scheduling)

	for _, option := range options {
		option(scheduling)
	}

	return scheduling
}

func (s *Scheduling) Run() error {
	scheduler, err := NewEngine()
	if err != nil {
		return err
	}

	for _, f := range s.Schedulers {
		f(scheduler)
	}

	scheduler.Start()
	defer scheduler.Shutdown()
	select {}
}
