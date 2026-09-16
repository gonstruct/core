package resolvers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/queueing/job"
	redisconnection "github.com/gonstruct/core/redis"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gonstruct/validation/env"
	"github.com/hibiken/asynq"
)

type Redis struct {
	Client asynq.Client
	Cache  *cache.Cache
}

func (e *Redis) Dispatch(constructor job.Job, delay ...time.Duration) error {
	payload, err := json.Marshal(constructor)
	if err != nil {
		return err
	}

	retention := time.Hour * 24
	if option, hasCustomRetention := constructor.(job.JobWithRetention); hasCustomRetention {
		retention = option.Retention()
	}

	options := []asynq.Option{
		asynq.Retention(retention),
	}

	debounceable, isDebounceable := constructor.(job.JobWithDebounce)

	if !isDebounceable && len(delay) > 0 && delay[0] > 0 {
		options = append(options, asynq.ProcessIn(delay[0]))
	}

	if option, hasCustomTries := constructor.(job.JobWithTries); hasCustomTries {
		options = append(options, asynq.MaxRetry(option.Tries()-1))
	}

	if option, hasCustomQueue := constructor.(job.JobWithQueue); hasCustomQueue {
		options = append(options, asynq.Queue(option.Queue()))
	}

	if option, hasWithoutOverlapping := constructor.(job.JobWithoutOverlapping); hasWithoutOverlapping {
		options = append(options, asynq.TaskID(option.Unique()))
	}

	if isDebounceable {
		dispatch := job.PrepareDebounceDispatch(context.Background(), e.Cache, debounceable)
		options = append(options, asynq.TaskID(dispatch.Token))
		if dispatch.Delay > 0 {
			options = append(options, asynq.ProcessIn(dispatch.Delay))
		}
	}

	_, err = e.Client.Enqueue(asynq.NewTask(constructor.Name(), payload), options...)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		log.Warn().Msgf("Skipping duplicate job: %s", constructor.Name())
		return nil
	}

	return err
}

//nolint:gocognit
func (e *Redis) Consume(jobs map[string]job.Job) error {
	handler := asynq.NewServeMux()

	queues := map[string]int{}
	for _, jobHandler := range jobs {
		if jobWithQueue, ok := jobHandler.(job.JobWithQueue); ok {
			queues[jobWithQueue.Queue()] = 1

			if jobWithPriority, ok := jobHandler.(job.JobWithPriority); ok {
				queues[jobWithQueue.Queue()] = jobWithPriority.Priority()
			}
		}

		_, isDebounceableHandler := jobHandler.(job.JobWithDebounce)
		_, hasOverlapGuard := jobHandler.(job.JobWithoutOverlapping)
		if isDebounceableHandler && hasOverlapGuard {
			log.Warn().Str("job", jobHandler.Name()).Msg("[queueing] JobWithDebounce and JobWithoutOverlapping should not be combined; debounce will win")
		}

		handler.HandleFunc(jobHandler.Name(), func(ctx context.Context, task *asynq.Task) error {
			constructed, err := jobHandler.Construct(task.Payload())
			if err != nil {
				return fmt.Errorf("[queueing] failed to construct job %s: %w", task.Type(), err)
			}

			if debounceable, isDebounceable := constructed.(job.JobWithDebounce); isDebounceable && e.Cache != nil {
				myToken, _ := asynq.GetTaskID(ctx)
				storedToken, found, lookupErr := e.Cache.Get(ctx, job.DebounceCachePrefix+debounceable.DebounceKey())
				switch {
				case lookupErr != nil:
					log.Warn().Ctx(ctx).Err(lookupErr).Str("task", task.Type()).Msg("[queueing] debounce: cache lookup failed, executing fail-open")
				case !found:
					log.Debug().Ctx(ctx).Str("task", task.Type()).Msg("[queueing] debounce: token missing, executing fail-open")
				case storedToken != myToken:
					log.Info().Ctx(ctx).Str("task", task.Type()).Msg("[queueing] debounce: superseded by newer dispatch, discarding")
					return asynq.RevokeTask // We silently drop the task instead of returning nil (nil means completed which isn't true)
				}
				ctx = job.WithDebounceToken(ctx, myToken)
			}

			if jobWithHydration, ok := constructed.(job.JobWithHydration); ok {
				if err := jobWithHydration.Hydrate(ctx); err != nil {
					return fmt.Errorf("[queueing] failed to hydrate job %s: %w", task.Type(), err)
				}
			}

			shouldContinue, err := job.Execute(ctx, constructed)
			if !shouldContinue {
				log.Warn().Ctx(ctx).Str("task", task.Type()).Msg("[queueing] middleware requested dontRelease, revoking task")
				return asynq.RevokeTask
			}

			if failErr, ok := job.AsFail(err); ok {
				log.Error().Ctx(ctx).Err(failErr).Str("task", task.Type()).Msg("[queueing] fail directive requested, skipping retries")
				return fmt.Errorf("%w: %v", asynq.SkipRetry, failErr)
			}

			return err
		})
	}

	return asynq.NewServer(RedisClient{}, asynq.Config{
		Concurrency: env.Number("QUEUE_CONCURRENCY"),
		Queues:      queues,
		RetryDelayFunc: func(retries int, err error, task *asynq.Task) time.Duration {
			if release, ok := job.AsRelease(err); ok {
				return release.Delay
			}

			if job, ok := jobs[task.Type()].(job.JobWithBackoff); ok {
				return job.Backoff(retries + 1)
			}

			return asynq.DefaultRetryDelayFunc(retries, err, task)
		},
		IsFailure: func(err error) bool {
			if release, ok := job.AsRelease(err); ok && release.Silent {
				return false
			}

			if strings.Contains(err.Error(), "sorry, too many clients already") {
				return false
			}

			return true
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, taskErr error) {
			constructed, err := jobs[task.Type()].Construct(task.Payload())
			if err != nil {
				log.Error().Ctx(ctx).Err(err).Msgf("[queueing] Failed to construct job %s: %v", task.Type(), err)
				return
			}

			if jobWithHydration, ok := constructed.(job.JobWithHydration); ok {
				if err := jobWithHydration.Hydrate(ctx); err != nil {
					log.Error().Ctx(ctx).Err(err).Msgf("[queueing] Failed to hydrate job %s: %v", task.Type(), err)
					return
				}
			}

			if jobWithFailing, ok := constructed.(job.JobWithFailure); ok {
				jobWithFailing.Failure(ctx, taskErr)
			}

			retried, _ := asynq.GetRetryCount(ctx)
			maxRetry, _ := asynq.GetMaxRetry(ctx)

			if retried >= maxRetry {
				log.Error().Ctx(ctx).Err(taskErr).Msgf("[queueing] Task %s failed after %d retries, calling failed", task.Type(), retried)

				constructed.Failed(ctx, taskErr)
			} else {
				log.Warn().Ctx(ctx).Err(taskErr).Msgf("[queueing] Task %s failed on attempt %d/%d, will retry", task.Type(), retried, maxRetry)
			}
		}),
	}).Run(handler)
}

type RedisClient struct{}

func (c RedisClient) MakeRedisClient() any {
	client, err := redisconnection.NewClient(env.String("QUEUE_REDIS_CONNECTION", redisconnection.DefaultConnection))
	if err != nil {
		panic(fmt.Errorf("[queueing] invalid Redis configuration: %w", err))
	}

	return client
}
