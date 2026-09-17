package queueing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gonstruct/core/queueing/job"

	"github.com/rs/zerolog/log"
)

type Queueing struct {
	Queues []func(*Engine)

	engine *Engine
	mu     sync.Mutex
}

func New(options ...Option) *Queueing {
	queueing := new(Queueing)

	for _, option := range options {
		option(queueing)
	}

	return queueing
}

func (q *Queueing) Run() error {
	if err := q.ensureEngine(); err != nil {
		return err
	}

	return q.engine.Consume(q.engine.jobs)
}

func (q *Queueing) Dispatch(ctx context.Context, current job.Job, delay ...time.Duration) error {
	if err := q.ensureEngine(); err != nil {
		return fmt.Errorf("[queueing] cannot dispatch job: %w", err)
	}

	prepared := current
	if dispatchPreparation, ok := current.(job.JobWithDispatchPreparation); ok {
		var err error
		prepared, err = dispatchPreparation.PrepareDispatch(ctx)
		if err != nil {
			return fmt.Errorf("[queueing] failed to prepare job dispatch: %w", err)
		}
		if prepared == nil {
			log.Info().Msgf("[queueing] skipped job dispatch: %s", current.Name())
			return nil
		}
	}

	log.Info().Msgf("[queueing] dispatching job: %s", prepared.Name())
	if err := q.engine.Dispatch(prepared, delay...); err != nil {
		log.Error().Err(err).Msgf("[queueing] failed to dispatch job: %s", prepared.Name())
		return fmt.Errorf("[queueing] failed to dispatch job: %w", err)
	}
	log.Info().Msgf("[queueing] dispatched job: %s", prepared.Name())
	return nil
}

func (q *Queueing) ensureEngine() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.engine != nil {
		return nil
	}

	engine, err := NewEngine()
	if err != nil {
		return err
	}

	for _, f := range q.Queues {
		f(engine)
	}

	q.engine = engine
	return nil
}
