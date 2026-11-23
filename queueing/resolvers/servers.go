package resolvers

import (
	"fmt"
	"maps"
	"slices"

	"github.com/gonstruct/core/config"
	"github.com/gonstruct/core/queueing/job"

	"github.com/rs/zerolog/log"
)

// DefaultQueue is where asynq puts a job that names no queue.
const DefaultQueue = "default"

// Server is one asynq server of the worker: a group's workers on the
// group's queue or, with no Group, the default workers on the queues of
// every job outside a group.
type Server struct {
	Group       string
	Concurrency int
	// Queues is the queues it takes jobs from, by weight.
	Queues map[string]int
}

// Servers is the worker's asynq servers: one for each group jobs are
// registered in, with the workers the config gives it, and one with the
// config's Concurrency for every other job, on the queue it names, weighted
// by Priority. groupOf is the group of each job in one, by job name. A group
// without workers is an error, not a silent share.
func Servers(jobs map[string]job.Job, groupOf map[string]string, queue config.Queue) ([]Server, error) {
	used := map[string]bool{}
	for _, group := range groupOf {
		used[group] = true
	}
	for group := range queue.Groups {
		if !used[group] {
			log.Warn().Msgf("[queueing] the config gives queue group %s workers, but no jobs are registered in it", group)
		}
	}

	servers := []Server{}
	for _, group := range slices.Sorted(maps.Keys(used)) {
		workers := queue.Groups[group].Concurrency
		if workers < 1 {
			return nil, fmt.Errorf("[queueing] queue group %s has no workers: give it a Concurrency in the config's Groups", group)
		}
		servers = append(servers, Server{Group: group, Concurrency: workers, Queues: map[string]int{group: 1}})
	}

	queues := map[string]int{}
	for name, each := range jobs {
		if groupOf[name] != "" {
			continue
		}
		queueName, weight := DefaultQueue, 1
		if named, ok := each.(job.JobWithQueue); ok {
			queueName = named.Queue()
		}
		if weighted, ok := each.(job.JobWithPriority); ok {
			weight = weighted.Priority()
		}
		if used[queueName] {
			return nil, fmt.Errorf("[queueing] job %s names queue %s, which is a group's: register it in the group", name, queueName)
		}
		queues[queueName] = max(queues[queueName], weight)
	}
	if len(queues) == 0 && len(servers) > 0 {
		return servers, nil
	}
	if queue.Concurrency < 1 {
		return nil, fmt.Errorf("[queueing] the default queue has no workers: give the config a Concurrency")
	}

	return append(servers, Server{Concurrency: queue.Concurrency, Queues: queues}), nil
}
