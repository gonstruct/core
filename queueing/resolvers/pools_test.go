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

// weighted is on a queue shared by weight.
type weighted struct {
	job.Queueable[*weighted]

	queue    string
	priority int
}

func (self *weighted) Name() string             { return "weighted:" + self.queue }
func (*weighted) Handle(context.Context) error  { return nil }
func (*weighted) Failed(context.Context, error) {}
func (self *weighted) Queue() string            { return self.queue }
func (self *weighted) Priority() int            { return self.priority }

// counted is on a queue with workers of its own.
type counted struct {
	job.Queueable[*counted]

	name        string
	queue       string
	concurrency int
}

func (self *counted) Name() string             { return self.name }
func (*counted) Handle(context.Context) error  { return nil }
func (*counted) Failed(context.Context, error) {}
func (self *counted) Queue() string            { return self.queue }
func (self *counted) Concurrency() int         { return self.concurrency }

func jobs(list ...job.Job) map[string]job.Job {
	byName := map[string]job.Job{}
	for _, each := range list {
		byName[each.Name()] = each
	}

	return byName
}

func TestPools(t *testing.T) {
	cases := []struct {
		name string
		jobs map[string]job.Job
		want []resolvers.Pool
	}{
		{
			name: "jobs without queues share the default queue",
			jobs: jobs(&plain{}),
			want: []resolvers.Pool{{Concurrency: 10, Queues: map[string]int{"default": 1}}},
		},
		{
			name: "weighted queues share the workers, and the default queue stays listened to",
			jobs: jobs(&plain{}, &weighted{queue: "mail", priority: 3}, &weighted{queue: "tracking", priority: 1}),
			want: []resolvers.Pool{{Concurrency: 10, Queues: map[string]int{"default": 1, "mail": 3, "tracking": 1}}},
		},
		{
			name: "a queue with a concurrency gets workers of its own",
			jobs: jobs(
				&counted{name: "launch", queue: "ci", concurrency: 150},
				&counted{name: "destroy", queue: "ci", concurrency: 150},
				&counted{name: "mail", queue: "background", concurrency: 100},
				&plain{},
			),
			want: []resolvers.Pool{
				{Concurrency: 100, Queues: map[string]int{"background": 1}},
				{Concurrency: 150, Queues: map[string]int{"ci": 1}},
				{Concurrency: 10, Queues: map[string]int{"default": 1}},
			},
		},
		{
			name: "with every queue counted there is no shared pool",
			jobs: jobs(&counted{name: "launch", queue: "ci", concurrency: 150}, &counted{name: "mail", queue: "default", concurrency: 100}),
			want: []resolvers.Pool{
				{Concurrency: 150, Queues: map[string]int{"ci": 1}},
				{Concurrency: 100, Queues: map[string]int{"default": 1}},
			},
		},
		{
			name: "jobs on one queue asking for different counts get the most",
			jobs: jobs(&counted{name: "launch", queue: "ci", concurrency: 150}, &counted{name: "destroy", queue: "ci", concurrency: 20}),
			want: []resolvers.Pool{{Concurrency: 150, Queues: map[string]int{"ci": 1}}},
		},
		{
			name: "a queue some jobs count and others weigh runs only on its own workers",
			jobs: jobs(&counted{name: "launch", queue: "ci", concurrency: 150}, &weighted{queue: "ci", priority: 5}),
			want: []resolvers.Pool{{Concurrency: 150, Queues: map[string]int{"ci": 1}}},
		},
	}

	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			if got := resolvers.Pools(each.jobs, 10); !reflect.DeepEqual(got, each.want) {
				t.Fatalf("got %+v, want %+v", got, each.want)
			}
		})
	}
}
