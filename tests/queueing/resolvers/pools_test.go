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
		name     string
		jobs     map[string]job.Job
		channels map[string]int
		want     []resolvers.Pool
	}{
		{
			name: "jobs without queues share the default queue",
			jobs: jobs(&plain{}),
			want: []resolvers.Pool{{Concurrency: 10, Queues: map[string]int{"default": 1}}},
		},
		{
			name: "queues that are not channels share the workers by priority, and the default queue stays listened to",
			jobs: jobs(&plain{}, &queued{name: "mail", queue: "mail", priority: 3}, &queued{name: "track", queue: "tracking"}),
			want: []resolvers.Pool{{Concurrency: 10, Queues: map[string]int{"default": 1, "mail": 3, "tracking": 1}}},
		},
		{
			name:     "a channel gets workers of its own, however its jobs are weighted",
			jobs:     jobs(&queued{name: "launch", queue: "ci", priority: 5}, &queued{name: "destroy", queue: "ci"}, &plain{}),
			channels: map[string]int{"ci": 150},
			want: []resolvers.Pool{
				{Concurrency: 150, Queues: map[string]int{"ci": 1}},
				{Concurrency: 10, Queues: map[string]int{"default": 1}},
			},
		},
		{
			name:     "with every queue a channel there is no shared pool",
			jobs:     jobs(&queued{name: "launch", queue: "ci"}, &queued{name: "mail", queue: "default"}),
			channels: map[string]int{"ci": 150, "default": 100},
			want: []resolvers.Pool{
				{Concurrency: 150, Queues: map[string]int{"ci": 1}},
				{Concurrency: 100, Queues: map[string]int{"default": 1}},
			},
		},
		{
			name:     "a channel of no workers is no channel",
			jobs:     jobs(&queued{name: "launch", queue: "ci"}),
			channels: map[string]int{"ci": 0},
			want:     []resolvers.Pool{{Concurrency: 10, Queues: map[string]int{"ci": 1}}},
		},
	}

	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			if got := resolvers.Pools(each.jobs, each.channels, 10); !reflect.DeepEqual(got, each.want) {
				t.Fatalf("got %+v, want %+v", got, each.want)
			}
		})
	}
}
