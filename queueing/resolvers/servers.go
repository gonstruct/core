package resolvers

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/gonstruct/core/config"
	"github.com/gonstruct/core/queueing/job"

	"github.com/rs/zerolog/log"
)

const DefaultQueue = "default"

// Server is one asynq server: a group's workers, or the default workers.
type Server struct {
	Name          string
	Concurrency   int
	CheckInterval time.Duration
	Queues        map[string]int // queue name -> priority
}

// Servers returns one server per group, reading only the group's queue, and
// a default server for the queues of every job outside a group.
func Servers(jobs map[string]job.Job, groups map[string]string, queue config.Queue) ([]Server, error) {
	grouped := map[string]bool{}
	for _, group := range groups {
		grouped[group] = true
	}
	for group := range queue.Groups {
		if !grouped[group] {
			log.Warn().Msgf("[queueing] group %s is configured but has no jobs", group)
		}
	}

	servers := []Server{}
	for _, group := range slices.Sorted(maps.Keys(grouped)) {
		settings := queue.Groups[group]
		if settings.Concurrency < 1 {
			return nil, fmt.Errorf("[queueing] group %s has no Concurrency in config.Queue.Groups", group)
		}
		servers = append(servers, Server{
			Name:          group,
			Concurrency:   settings.Concurrency,
			CheckInterval: settings.CheckInterval,
			Queues:        map[string]int{group: 1},
		})
	}

	queues := map[string]int{}
	for name, each := range jobs {
		if _, ok := groups[name]; ok {
			continue
		}
		queueName, priority := DefaultQueue, 1
		if named, ok := each.(job.JobWithQueue); ok {
			queueName = named.Queue()
		}
		if weighted, ok := each.(job.JobWithPriority); ok {
			priority = weighted.Priority()
		}
		if grouped[queueName] {
			return nil, fmt.Errorf("[queueing] job %s uses group %s's queue but is not in the group", name, queueName)
		}
		queues[queueName] = max(queues[queueName], priority)
	}
	if len(queues) == 0 && len(servers) > 0 {
		return servers, nil
	}
	if queue.Concurrency < 1 {
		return nil, fmt.Errorf("[queueing] config.Queue has no Concurrency for jobs outside a group")
	}

	return append(servers, Server{
		Name:          DefaultQueue,
		Concurrency:   queue.Concurrency,
		CheckInterval: queue.CheckInterval,
		Queues:        queues,
	}), nil
}
