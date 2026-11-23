package resolvers_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gonstruct/core/config"
	"github.com/gonstruct/core/queueing/job"
	"github.com/gonstruct/core/queueing/resolvers"
)

// plain names no queue.
type plain struct{ job.Queueable[*plain] }

func (*plain) Name() string                  { return "plain" }
func (*plain) Handle(context.Context) error  { return nil }
func (*plain) Failed(context.Context, error) {}

// queued names a queue, the default one when none is set, weighted when
// priority is.
type queued struct {
	job.Queueable[*queued]

	name     string
	queue    string
	priority int
}

func (self *queued) Name() string             { return self.name }
func (*queued) Handle(context.Context) error  { return nil }
func (*queued) Failed(context.Context, error) {}
func (self *queued) Priority() int            { return max(1, self.priority) }

func (self *queued) Queue() string {
	if self.queue == "" {
		return "default"
	}

	return self.queue
}

func jobs(list ...job.Job) map[string]job.Job {
	byName := map[string]job.Job{}
	for _, each := range list {
		byName[each.Name()] = each
	}

	return byName
}

func TestServers(t *testing.T) {
	cases := []struct {
		name   string
		jobs   map[string]job.Job
		groups map[string]string
		config config.Queue
		want   []resolvers.Server
	}{
		{
			name:   "jobs outside a group share the default queue's workers",
			jobs:   jobs(&plain{}),
			config: config.Queue{Concurrency: 10},
			want:   []resolvers.Server{{Concurrency: 10, Queues: map[string]int{"default": 1}}},
		},
		{
			name:   "jobs outside a group naming queues share them by priority",
			jobs:   jobs(&plain{}, &queued{name: "mail", queue: "mail", priority: 3}, &queued{name: "track", queue: "tracking"}),
			config: config.Queue{Concurrency: 10},
			want:   []resolvers.Server{{Concurrency: 10, Queues: map[string]int{"default": 1, "mail": 3, "tracking": 1}}},
		},
		{
			name:   "a group gets workers of its own, on its queue",
			jobs:   jobs(&queued{name: "launch"}, &queued{name: "destroy"}, &plain{}),
			groups: map[string]string{"launch": "ci", "destroy": "ci"},
			config: config.Queue{Concurrency: 100, Groups: map[string]config.QueueGroup{"ci": {Concurrency: 150}}},
			want: []resolvers.Server{
				{Group: "ci", Concurrency: 150, Queues: map[string]int{"ci": 1}},
				{Concurrency: 100, Queues: map[string]int{"default": 1}},
			},
		},
		{
			name:   "with every job in a group the default queue has no workers",
			jobs:   jobs(&queued{name: "launch"}),
			groups: map[string]string{"launch": "ci"},
			config: config.Queue{Groups: map[string]config.QueueGroup{"ci": {Concurrency: 150}}},
			want:   []resolvers.Server{{Group: "ci", Concurrency: 150, Queues: map[string]int{"ci": 1}}},
		},
		{
			name:   "each server looks for work as often as its config says",
			jobs:   jobs(&queued{name: "launch"}, &plain{}),
			groups: map[string]string{"launch": "ci"},
			config: config.Queue{
				Concurrency:   100,
				CheckInterval: time.Second,
				Groups:        map[string]config.QueueGroup{"ci": {Concurrency: 150, CheckInterval: 50 * time.Millisecond}},
			},
			want: []resolvers.Server{
				{Group: "ci", Concurrency: 150, CheckInterval: 50 * time.Millisecond, Queues: map[string]int{"ci": 1}},
				{Concurrency: 100, CheckInterval: time.Second, Queues: map[string]int{"default": 1}},
			},
		},
	}

	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			got, err := resolvers.Servers(each.jobs, each.groups, each.config)
			if err != nil || !reflect.DeepEqual(got, each.want) {
				t.Fatalf("got %+v (%v), want %+v", got, err, each.want)
			}
		})
	}
}

func TestServersRefuse(t *testing.T) {
	cases := []struct {
		name   string
		jobs   map[string]job.Job
		groups map[string]string
		config config.Queue
	}{
		{
			name:   "a group the config gives no workers",
			jobs:   jobs(&queued{name: "launch"}),
			groups: map[string]string{"launch": "ci"},
			config: config.Queue{Concurrency: 10},
		},
		{
			name:   "a job outside the group naming the group's queue",
			jobs:   jobs(&queued{name: "launch"}, &queued{name: "stray", queue: "ci"}),
			groups: map[string]string{"launch": "ci"},
			config: config.Queue{Concurrency: 10, Groups: map[string]config.QueueGroup{"ci": {Concurrency: 5}}},
		},
		{
			name:   "jobs outside a group with no default workers",
			jobs:   jobs(&plain{}),
			config: config.Queue{},
		},
	}

	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			if servers, err := resolvers.Servers(each.jobs, each.groups, each.config); err == nil {
				t.Fatalf("expected an error, got %+v", servers)
			}
		})
	}
}
