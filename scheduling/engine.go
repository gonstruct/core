package scheduling

import (
	"context"
	"fmt"
	"os"
	"slices"
	"time"

	otelcore "github.com/gonstruct/core/otel"
	"github.com/gonstruct/core/scheduling/command"

	"github.com/rs/zerolog/log"

	"github.com/go-co-op/gocron/v2"
)

func NewEngine() (*Engine, error) {
	scheduler, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))
	if err != nil {
		return nil, err
	}

	return &Engine{
		Scheduler: scheduler,
	}, nil
}

func NewSynchronousEngine() *Engine {
	return &Engine{
		synchronous: true,
	}
}

type Engine struct {
	gocron.Scheduler

	synchronous bool
	Commands    []command.Command
}

func (e *Engine) Command(cmd command.Command, options ...command.Option) {
	log.Info().Msgf("[scheduling] registering command %s", cmd.Name())

	if e.synchronous {
		log.Info().Msgf("[scheduling] scheduling command %s to run synchronously", cmd.Name())
		e.Commands = append(e.Commands, cmd)
		return
	}

	opts := command.NewOptions(options...)

	if len(opts.Environments) > 0 && !slices.Contains(opts.Environments, os.Getenv("APP_ENV")) {
		log.Info().Msgf("[scheduling] skipping command %s for environment %s", cmd.Name(), os.Getenv("APP_ENV"))
		return
	}

	if opts.Timezone != "" && opts.Timezone != "UTC" {
		opts.Crontab = fmt.Sprintf("CRON_TZ=%s %s", opts.Timezone, opts.Crontab)
	}

	taskOptions := []gocron.JobOption{
		gocron.WithName(cmd.Name()),
	}

	if opts.WithoutOverlapping {
		taskOptions = append(taskOptions, gocron.WithSingletonMode(gocron.LimitModeReschedule))
	}

	if opts.RunImmediately {
		taskOptions = append(taskOptions, gocron.JobOption(gocron.WithStartImmediately()))
	}

	log.Info().Msgf("[scheduling] scheduling command %s with crontab '%s'", cmd.Name(), opts.Crontab)
	_, err := e.NewJob(gocron.CronJob(opts.Crontab, opts.WithSeconds), gocron.NewTask(e.wrapTaskHandler(cmd, opts)), taskOptions...)
	if err != nil {
		panic(fmt.Errorf("[scheduling] failed to schedule command %s: %w", cmd.Name(), err))
	}
}

//nolint:gocognit
func (e *Engine) wrapTaskHandler(cmd command.Command, options *command.Options) func(context.Context) {
	return func(context context.Context) {
		if options.When != nil && !options.When() {
			log.Info().Msgf("[scheduling] skipping command %s because When condition is not met", cmd.Name())
			return
		}

		if options.Skip != nil && options.Skip() {
			log.Info().Msgf("[scheduling] skipping command %s because Skip condition is met", cmd.Name())
			return
		}

		if options.Between != [2]string{} {
			now := time.Now().Format("15:04")
			if now < options.Between[0] || now > options.Between[1] {
				log.Info().Msgf("[scheduling] skipping command %s because current time is not between %s and %s", cmd.Name(), options.Between[0], options.Between[1])
				return
			}
		}

		if options.UnlessBetween != [2]string{} {
			now := time.Now().Format("15:04")
			if now >= options.UnlessBetween[0] && now <= options.UnlessBetween[1] {
				//nolint:lll
				log.Info().Msgf("[scheduling] skipping command %s because current time is between %s and %s", cmd.Name(), options.UnlessBetween[0], options.UnlessBetween[1])
				return
			}
		}

		if options.Weekdays && (time.Now().Weekday() == time.Saturday || time.Now().Weekday() == time.Sunday) {
			log.Info().Msgf("[scheduling] skipping command %s because today is not a weekday", cmd.Name())
			return
		}

		if options.Weekends && (time.Now().Weekday() != time.Saturday && time.Now().Weekday() != time.Sunday) {
			log.Info().Msgf("[scheduling] skipping command %s because today is not a weekend", cmd.Name())
			return
		}

		if len(options.DaysOfWeek) > 0 && !slices.Contains(options.DaysOfWeek, time.Now().Weekday()) {
			log.Info().Msgf("[scheduling] skipping command %s because today is not in the specified days of the week", cmd.Name())
			return
		}

		log.Info().Msgf("[scheduling] executing command %s", cmd.Name())

		context, finish := otelcore.Instance().StartCommand(context, cmd.Name())

		// A panic must still end the span, and end it as a failure. finish is what
		// calls span.End(), so without this the run that crashed is the one that
		// leaves no trace at all — the exact inverse of what you want. The panic
		// is re-raised so the caller's own recovery is unchanged.
		defer func() {
			if recovered := recover(); recovered != nil {
				finish(fmt.Errorf("panic: %v", recovered))

				panic(recovered)
			}
		}()

		err := cmd.Handle(context)
		finish(err)

		if err != nil {
			log.Error().Err(err).Msgf("[scheduling] command %s failed", cmd.Name())
		}
	}
}
