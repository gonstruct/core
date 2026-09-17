package resolvers

import (
	"context"
	"time"

	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/queueing/job"

	"github.com/rs/zerolog/log"
)

type Sync struct {
	Cache *cache.Cache
}

func (e *Sync) Dispatch(current job.Job, delay ...time.Duration) error {
	debounceable, isDebounceable := current.(job.JobWithDebounce)
	debounceToken := ""
	if isDebounceable && e.Cache != nil {
		dispatch := job.PrepareDebounceDispatch(context.Background(), e.Cache, debounceable)
		debounceToken = dispatch.Token
		// Debounce schedules the job itself; ignore any caller-provided delay.
		if dispatch.Delay > 0 {
			delay = []time.Duration{dispatch.Delay}
		} else {
			delay = nil
		}
	}

	var schedule func(time.Duration) error

	handle := func() error {
		ctx := context.Background()

		if isDebounceable && e.Cache != nil {
			storedToken, found, lookupErr := e.Cache.Get(ctx, job.DebounceCachePrefix+debounceable.DebounceKey())
			switch {
			case lookupErr != nil:
				log.Warn().Err(lookupErr).Str("job", current.Name()).Msg("[queueing] debounce: cache lookup failed, executing fail-open")
			case !found:
				log.Debug().Str("job", current.Name()).Msg("[queueing] debounce: token missing, executing fail-open")
			case storedToken != debounceToken:
				log.Info().Str("job", current.Name()).Msg("[queueing] debounce: superseded by newer dispatch, discarding")
				return nil
			}
			ctx = job.WithDebounceToken(ctx, debounceToken)
		}

		if jobWithHydration, ok := current.(job.JobWithHydration); ok {
			if err := jobWithHydration.Hydrate(ctx); err != nil {
				return err
			}
		}

		shouldContinue, err := job.Execute(ctx, current)
		if !shouldContinue {
			log.Warn().Str("job", current.Name()).Msg("[queueing] middleware requested dontRelease, dropping task")
			return err
		}

		if release, ok := job.AsRelease(err); ok {
			return schedule(release.Delay)
		}

		if err != nil {
			log.Error().Err(err).Msgf("Job %s failed", current.Name())
			current.Failed(ctx, err)
		}

		return nil
	}

	schedule = func(wait time.Duration) error {
		if wait > 0 {
			go func() {
				log.Info().Msgf("[queueing] delaying job %s for %s", current.Name(), wait)
				time.Sleep(wait)
				if err := handle(); err != nil {
					log.Error().Err(err).Msgf("Failed to execute delayed job %s", current.Name())
				}
			}()
			return nil
		}

		return handle()
	}

	if len(delay) > 0 {
		return schedule(delay[0])
	}

	return schedule(0)
}

func (e *Sync) Consume(jobs map[string]job.Job) error {
	return nil
}
