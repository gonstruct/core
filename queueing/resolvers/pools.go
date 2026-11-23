package resolvers

import (
	"fmt"
	"sort"
	"time"

	"github.com/gonstruct/core/config"
	"github.com/gonstruct/core/queueing/job"

	"github.com/rs/zerolog/log"
)

// DefaultQueue is where asynq puts a job that names no queue.
const DefaultQueue = "default"

// DefaultPool is the name of the pool of every job outside a group.
const DefaultPool = "default"

// Pool is what one asynq server of the worker runs: Concurrency workers
// taking from Queues, by weight.
type Pool struct {
	Name          string
	Concurrency   int
	CheckInterval time.Duration
	Queues        map[string]int
}

// Pools is a pool for each group, on its queue with the workers the config
// gives it, and one of the config's Concurrency for every job outside a
// group: on the queue it names, weighted by Priority, or the default one. A
// group the config gives no workers is an error, not a silent share.
func Pools(jobs map[string]job.Job, groups map[string]string, queue config.Queue) ([]Pool, error) {
	grouped := map[string]bool{}
	for _, group := range groups {
		grouped[group] = true
	}
	names := make([]string, 0, len(grouped))
	for group := range grouped {
		names = append(names, group)
	}
	sort.Strings(names)

	pools := make([]Pool, 0, len(names)+1)
	for _, group := range names {
		workers := queue.Groups[group].Concurrency
		if workers < 1 {
			return nil, fmt.Errorf("[queueing] queue group %s has no workers: give it a Concurrency in the config's Groups", group)
		}
		pools = append(pools, Pool{
			Name:          group,
			Concurrency:   workers,
			CheckInterval: queue.Groups[group].CheckInterval,
			Queues:        map[string]int{group: 1},
		})
	}
	for group := range queue.Groups {
		if !grouped[group] {
			log.Warn().Msgf("[queueing] the config gives queue group %s workers, but no jobs are registered in it", group)
		}
	}

	sharing := map[string]int{}
	for name, each := range jobs {
		if _, inGroup := groups[name]; inGroup {
			continue
		}
		queueName, weight := DefaultQueue, 1
		if named, ok := each.(job.JobWithQueue); ok {
			queueName = named.Queue()
		}
		if weighted, ok := each.(job.JobWithPriority); ok {
			weight = weighted.Priority()
		}
		if grouped[queueName] {
			return nil, fmt.Errorf("[queueing] job %s names queue %s, which is a group's: register it in the group", name, queueName)
		}
		sharing[queueName] = max(sharing[queueName], weight)
	}
	if len(sharing) > 0 || len(pools) == 0 {
		if queue.Concurrency < 1 {
			return nil, fmt.Errorf("[queueing] the default queue has no workers: give the config a Concurrency")
		}
		pools = append(pools, Pool{
			Name:          DefaultPool,
			Concurrency:   queue.Concurrency,
			CheckInterval: queue.CheckInterval,
			Queues:        sharing,
		})
	}

	return pools, nil
}
