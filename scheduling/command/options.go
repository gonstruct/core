package command

import "time"

func NewOptions(options ...Option) *Options {
	opts := &Options{
		Timezone: "UTC",
	}

	for _, option := range options {
		option(opts)
	}

	return opts
}

type Options struct {
	Crontab     string
	WithSeconds bool
	Timezone    string

	// Constraints
	Weekdays      bool
	Weekends      bool
	DaysOfWeek    []time.Weekday
	Between       [2]string
	UnlessBetween [2]string

	// Execution controls
	WithoutOverlapping bool
	// OnOneServer        bool
	// RunInBackground    bool
	// EvenInMaintenance bool
	RunImmediately bool

	// Conditions
	When         func() bool
	Skip         func() bool
	Environments []string

	// Output
	SendOutputTo      string
	AppendOutputTo    string
	EmailOutputTo     string
	EmailOutputOnFail string
}

type Option func(*Options)
