package queueing_test

import (
	"context"
	"strings"
	"testing"

	"github.com/gonstruct/core/cache"
	"github.com/gonstruct/core/config"
	"github.com/gonstruct/core/queueing"
	"github.com/gonstruct/core/queueing/job"
)

type launch struct{ job.Queueable[*launch] }

func (*launch) Name() string                  { return "launch" }
func (*launch) Handle(context.Context) error  { return nil }
func (*launch) Failed(context.Context, error) {}

type mail struct{ job.Queueable[*mail] }

func (*mail) Name() string                  { return "mail" }
func (*mail) Handle(context.Context) error  { return nil }
func (*mail) Failed(context.Context, error) {}

func TestQueueingNeedsAConfig(t *testing.T) {
	err := queueing.New(queueing.WithQueues(func(*queueing.Engine) {})).Dispatch(t.Context(), &mail{})
	if err == nil || !strings.Contains(err.Error(), "no config") {
		t.Fatalf("expected no config, got %v", err)
	}
}

func TestAnUnknownDriverIsRefused(t *testing.T) {
	if _, err := queueing.NewEngine(config.Queue{Driver: "carrier-pigeon"}, cache.New()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestAJobGoesOnItsGroupsQueue(t *testing.T) {
	engine, err := queueing.NewEngine(config.Queue{Driver: config.SynchronousQueueDriver}, cache.New())
	if err != nil {
		t.Fatal(err)
	}
	engine.Group("ci", func(group *queueing.Group) {
		group.Job(&launch{})
	})
	engine.Job(&mail{})

	if queue := engine.QueueOf(&launch{}); queue != "ci" {
		t.Fatalf("expected launch on ci, got %q", queue)
	}
	if queue := engine.QueueOf(&mail{}); queue != "" {
		t.Fatalf("expected mail on its own queue, got %q", queue)
	}
}
