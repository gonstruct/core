package command

import "time"

func Weekdays() func(*Options) { return func(o *Options) { o.Weekdays = true } }
func Weekends() func(*Options) { return func(o *Options) { o.Weekends = true } }

func Mondays() func(*Options) {
	return func(o *Options) { o.DaysOfWeek = append(o.DaysOfWeek, time.Monday) }
}

func Tuesdays() func(*Options) {
	return func(o *Options) { o.DaysOfWeek = append(o.DaysOfWeek, time.Tuesday) }
}

func Wednesdays() func(*Options) {
	return func(o *Options) { o.DaysOfWeek = append(o.DaysOfWeek, time.Wednesday) }
}

func Thursdays() func(*Options) {
	return func(o *Options) { o.DaysOfWeek = append(o.DaysOfWeek, time.Thursday) }
}

func Fridays() func(*Options) {
	return func(o *Options) { o.DaysOfWeek = append(o.DaysOfWeek, time.Friday) }
}

func Saturdays() func(*Options) {
	return func(o *Options) { o.DaysOfWeek = append(o.DaysOfWeek, time.Saturday) }
}

func Sundays() func(*Options) {
	return func(o *Options) { o.DaysOfWeek = append(o.DaysOfWeek, time.Sunday) }
}

func Between(start, end string) func(*Options) {
	return func(o *Options) { o.Between = [2]string{start, end} }
}

func UnlessBetween(start, end string) func(*Options) {
	return func(o *Options) { o.UnlessBetween = [2]string{start, end} }
}

// WithoutOverlapping is an execution control.
func WithoutOverlapping() func(*Options) {
	return func(o *Options) { o.WithoutOverlapping = true }
}

func RunImmediately() func(*Options) {
	return func(o *Options) { o.RunImmediately = true }
}

// func OnOneServer() func(*Options) {
// 	return func(o *Options) { o.OnOneServer = true }
// }

// func RunInBackground() func(*Options) {
// 	return func(o *Options) { o.RunInBackground = true }
// }

// func EvenInMaintenanceMode() func(*Options) {
// 	return func(o *Options) { o.EvenInMaintenance = true }
// }

// When is a condition.
func When(fn func() bool) func(*Options) {
	return func(o *Options) { o.When = fn }
}

func Skip(fn func() bool) func(*Options) {
	return func(o *Options) { o.Skip = fn }
}

func Environment[S string](env ...string) func(*Options) {
	return func(o *Options) { o.Environments = env }
}

// SendOutputTo is an output control.
func SendOutputTo(path string) func(*Options) {
	return func(o *Options) { o.SendOutputTo = path }
}

func AppendOutputTo(path string) func(*Options) {
	return func(o *Options) { o.AppendOutputTo = path }
}

func EmailOutputTo(address string) func(*Options) {
	return func(o *Options) { o.EmailOutputTo = address }
}

func EmailOutputOnFailure(address string) func(*Options) {
	return func(o *Options) { o.EmailOutputOnFail = address }
}
