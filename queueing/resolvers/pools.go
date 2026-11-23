package resolvers

import (
	"github.com/gonstruct/core/queueing/job"
)

// DefaultQueue is where asynq puts a job that names no queue.
const DefaultQueue = "default"

// SharedPool is the name of the pool of the queues no supervisor reads.
const SharedPool = "shared"

// Supervisor is a group of workers, as in Laravel Horizon: up to Workers
// jobs at once, from its Queues, in the order given.
type Supervisor struct {
	Name    string
	Workers int
	Queues  []string
}

// Pool is what one asynq server of the worker runs: Concurrency workers
// taking from Queues, by weight, or highest first when Strict.
type Pool struct {
	Name        string
	Concurrency int
	Queues      map[string]int
	Strict      bool
}

// Pools is a pool for each supervisor with workers, and one of shared
// workers for every other queue the jobs name, weighted by Priority. A job
// that names no queue is on asynq's default one, which the shared pool then
// reads unless a supervisor does, or it would never run.
func Pools(jobs map[string]job.Job, supervisors []Supervisor, shared int) []Pool {
	supervised := map[string]bool{}
	pools := make([]Pool, 0, len(supervisors)+1)
	for _, supervisor := range supervisors {
		if supervisor.Workers < 1 || len(supervisor.Queues) == 0 {
			continue
		}
		// The first queue weighs the most; strict, it is emptied first.
		queues := make(map[string]int, len(supervisor.Queues))
		for index, queue := range supervisor.Queues {
			queues[queue] = len(supervisor.Queues) - index
			supervised[queue] = true
		}
		pools = append(pools, Pool{Name: supervisor.Name, Concurrency: supervisor.Workers, Queues: queues, Strict: len(queues) > 1})
	}

	sharing := map[string]int{}
	for _, each := range jobs {
		queue, weight := DefaultQueue, 1
		if named, ok := each.(job.JobWithQueue); ok {
			queue = named.Queue()
		}
		if weighted, ok := each.(job.JobWithPriority); ok {
			weight = weighted.Priority()
		}
		if !supervised[queue] {
			sharing[queue] = max(sharing[queue], weight)
		}
	}
	if len(sharing) > 0 || len(pools) == 0 {
		pools = append(pools, Pool{Name: SharedPool, Concurrency: shared, Queues: sharing})
	}

	return pools
}
