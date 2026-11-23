package resolvers_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/gonstruct/core/queueing/job"
	"github.com/gonstruct/core/queueing/resolvers"
)

// plain names no queue.
type plain struct{ job.Queueable[*plain] }

func (*plain) Name() string                  { return "plain" }
func (*plain) Handle(context.Context) error  { return nil }
func (*plain) Failed(context.Context, error) {}

// queued is on a queue, weighted when priority is set.
type queued struct {
	job.Queueable[*queued]

	name     string
	queue    string
	priority int
}

func (self *queued) Name() string             { return self.name }
func (*queued) Handle(context.Context) error  { return nil }
func (*queued) Failed(context.Context, error) {}
func (self *queued) Queue() string            { return self.queue }
func (self *queued) Priority() int            { return max(1, self.priority) }

func jobs(list ...job.Job) map[string]job.Job {
	byName := map[string]job.Job{}
	for _, each := range list {
		byName[each.Name()] = each
	}

	return byName
}

func TestPools(t *testing.T) {
	cases := []struct {
		name        string
		jobs        map[string]job.Job
		supervisors []resolvers.Supervisor
		want        []resolvers.Pool
	}{
		{
			name: "jobs without queues share the default queue",
			jobs: jobs(&plain{}),
			want: []resolvers.Pool{{Name: "shared", Concurrency: 10, Queues: map[string]int{"default": 1}}},
		},
		{
			name: "queues no supervisor reads share the workers by priority, the default queue among them",
			jobs: jobs(&plain{}, &queued{name: "mail", queue: "mail", priority: 3}, &queued{name: "track", queue: "tracking"}),
			want: []resolvers.Pool{{Name: "shared", Concurrency: 10, Queues: map[string]int{"default": 1, "mail": 3, "tracking": 1}}},
		},
		{
			name:        "a supervisor's workers take only from its queue, whatever its jobs weigh",
			jobs:        jobs(&queued{name: "launch", queue: "ci", priority: 5}, &queued{name: "destroy", queue: "ci"}, &plain{}),
			supervisors: []resolvers.Supervisor{{Name: "ci", Workers: 150, Queues: []string{"ci"}}},
			want: []resolvers.Pool{
				{Name: "ci", Concurrency: 150, Queues: map[string]int{"ci": 1}},
				{Name: "shared", Concurrency: 10, Queues: map[string]int{"default": 1}},
			},
		},
		{
			name: "with every queue supervised there are no shared workers",
			jobs: jobs(&queued{name: "launch", queue: "ci"}, &queued{name: "mail", queue: "default"}, &plain{}),
			supervisors: []resolvers.Supervisor{
				{Name: "ci", Workers: 150, Queues: []string{"ci"}},
				{Name: "background", Workers: 100, Queues: []string{"default"}},
			},
			want: []resolvers.Pool{
				{Name: "ci", Concurrency: 150, Queues: map[string]int{"ci": 1}},
				{Name: "background", Concurrency: 100, Queues: map[string]int{"default": 1}},
			},
		},
		{
			name:        "a supervisor of several queues empties them in order",
			jobs:        jobs(&queued{name: "urgent", queue: "high"}, &queued{name: "mail", queue: "default"}),
			supervisors: []resolvers.Supervisor{{Name: "main", Workers: 20, Queues: []string{"high", "default"}}},
			want:        []resolvers.Pool{{Name: "main", Concurrency: 20, Queues: map[string]int{"high": 2, "default": 1}, Strict: true}},
		},
		{
			name:        "a supervisor of no workers reads nothing",
			jobs:        jobs(&queued{name: "launch", queue: "ci"}),
			supervisors: []resolvers.Supervisor{{Name: "ci", Workers: 0, Queues: []string{"ci"}}},
			want:        []resolvers.Pool{{Name: "shared", Concurrency: 10, Queues: map[string]int{"ci": 1}}},
		},
	}

	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			if got := resolvers.Pools(each.jobs, each.supervisors, 10); !reflect.DeepEqual(got, each.want) {
				t.Fatalf("got %+v, want %+v", got, each.want)
			}
		})
	}
}
