package resolvers

import (
	"sort"

	"github.com/gonstruct/core/queueing/job"

	"github.com/rs/zerolog/log"
)

// DefaultQueue is where asynq puts a job that names no queue.
const DefaultQueue = "default"

// Pool is a set of queues the worker takes jobs from with workers of their
// own: Concurrency of them, shared between the queues by weight.
type Pool struct {
	Concurrency int
	Queues      map[string]int
}

// Pools splits the queues the jobs name between the worker's pools. A queue
// with a concurrency of its own (job.JobWithConcurrency) is a pool by
// itself; the rest share one of shared workers, weighted by Priority. A job
// that names no queue is on asynq's default one, which the shared pool then
// listens to, or it would never run.
func Pools(jobs map[string]job.Job, shared int) []Pool {
	sharing := map[string]int{}
	own := map[string]int{}
	for _, each := range jobs {
		named, ok := each.(job.JobWithQueue)
		if !ok {
			sharing[DefaultQueue] = max(sharing[DefaultQueue], 1)

			continue
		}
		queue := named.Queue()
		if counted, ok := each.(job.JobWithConcurrency); ok && counted.Concurrency() > 0 {
			if own[queue] != 0 && own[queue] != counted.Concurrency() {
				log.Warn().Str("queue", queue).Msgf("[queueing] jobs on %s ask for %d and %d workers; it gets the most", queue, own[queue], counted.Concurrency())
			}
			own[queue] = max(own[queue], counted.Concurrency())

			continue
		}
		weight := 1
		if weighted, ok := each.(job.JobWithPriority); ok {
			weight = weighted.Priority()
		}
		sharing[queue] = max(sharing[queue], weight)
	}

	pools := make([]Pool, 0, len(own)+1)
	for queue, concurrency := range own {
		// A queue some jobs give workers of their own runs only on those.
		delete(sharing, queue)
		pools = append(pools, Pool{Concurrency: concurrency, Queues: map[string]int{queue: 1}})
	}
	sort.Slice(pools, func(left, right int) bool { return firstQueue(pools[left]) < firstQueue(pools[right]) })
	if len(sharing) > 0 || len(pools) == 0 {
		pools = append(pools, Pool{Concurrency: shared, Queues: sharing})
	}

	return pools
}

func firstQueue(pool Pool) string {
	for queue := range pool.Queues {
		return queue
	}

	return ""
}
