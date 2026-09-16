package console

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

// Writer renders records for a terminal. It reads the section, task and
// request markers below and lays each one out differently.
func Writer() io.Writer {
	return &ConsoleWriter{out: os.Stderr}
}

// Section prints a heading. Used to break long CLI output into blocks.
func Section(message string) {
	log.Info().Bool("section", true).Msg(message)
}

// Task runs a step and reports how long it took, then returns its error so the
// caller still decides what to do about it.
func Task(description string, run func() error) error {
	start := time.Now()
	err := run()

	event := log.Info()
	if err != nil {
		event = log.Error().Err(err)
	}

	event.Bool("task", true).Dur("duration", time.Since(start)).Msg(description)

	return err
}

// Request prints one handled HTTP request. It takes the context so the line
// lands on the request's own trace.
func Request(context context.Context, method string, path string, status int, elapsed time.Duration, requestID string) {
	log.Info().
		Ctx(context).
		Bool("request", true).
		Str("request_id", requestID).
		Str("method", method).
		Str("path", path).
		Int("status", status).
		Dur("duration", elapsed).
		Msg(method + " " + path)
}
