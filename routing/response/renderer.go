package response

import (
	"errors"
	"github.com/gonstruct/core/routing/response/exception"
)

// ExceptionRenderer returns a response for an error it recognizes, or nil to
// let the next renderer handle it.
type ExceptionRenderer func(exception exception.Exception) Response

var exceptionRenderers []ExceptionRenderer

func RenderException(renderer ExceptionRenderer) {
	exceptionRenderers = append(exceptionRenderers, renderer)
}

// RenderExceptionIs registers a renderer for errors matched by errors.Is.
func RenderExceptionIs(target error, renderer ExceptionRenderer) {
	RenderException(func(input exception.Exception) Response {
		if !errors.Is(input.Err, target) {
			return nil
		}

		return renderer(input)
	})
}

// RenderExceptionAs registers a renderer for errors matched by errors.As.
func RenderExceptionAs[T error](renderer func(err T, exception exception.Exception) Response) {
	RenderException(func(input exception.Exception) Response {
		var target T
		if !errors.As(input.Err, &target) {
			return nil
		}

		return renderer(target, input)
	})
}
