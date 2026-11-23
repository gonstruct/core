package resolvers

import (
	"sort"

	"github.com/gonstruct/core/queueing/job"
)

// DefaultQueue is where asynq puts a job that names no queue.
const DefaultQueue = "default"

// Pool is a set of queues the worker takes jobs from with workers of their
// own: Concurrency of them, shared between the queues by weight.
type Pool struct {
	Concurrency int
	Queues      map[string]int
}

// Pools splits the worker into its pools: each channel, a queue with
// workers of its own, by itself; and every other queue the jobs name in one
// of shared workers, weighted by Priority. A job that names no queue is on
// asynq's default one, which the shared pool then listens to, or it would
// never run.
func Pools(jobs map[string]job.Job, channels map[string]int, shared int) []Pool {
	sharing := map[string]int{}
	for _, each := range jobs {
		named, ok := each.(job.JobWithQueue)
		if !ok {
			sharing[DefaultQueue] = max(sharing[DefaultQueue], 1)

			continue
		}
		// A channel of no workers is no channel: its jobs share.
		if channels[named.Queue()] > 0 {
			continue
		}
		weight := 1
		if weighted, ok := each.(job.JobWithPriority); ok {
			weight = weighted.Priority()
		}
		sharing[named.Queue()] = max(sharing[named.Queue()], weight)
	}

	names := make([]string, 0, len(channels))
	for queue, concurrency := range channels {
		if concurrency > 0 {
			names = append(names, queue)
		}
	}
	sort.Strings(names)

	pools := make([]Pool, 0, len(names)+1)
	for _, queue := range names {
		pools = append(pools, Pool{Concurrency: channels[queue], Queues: map[string]int{queue: 1}})
	}
	if len(sharing) > 0 || len(pools) == 0 {
		pools = append(pools, Pool{Concurrency: shared, Queues: sharing})
	}

	return pools
}
