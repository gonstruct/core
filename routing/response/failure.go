package response

import (
	"github.com/gonstruct/core/routing/response/exception"
	"net/http"
)

func Errors(errors map[string]string, options ...responseOption) Response {
	defaultOptions := []responseOption{
		WithErrors(errors),
		WithStatusCode(http.StatusBadRequest),
	}

	return JSON(append(defaultOptions, options...)...)
}

// Exception renders an error using the registered exception renderers. Use it
// when the call site can provide neutral context such as the affected subject.
func Exception(err error, options ...exception.Option) Response {
	return renderError(exception.New(err, options...), nil)
}

// Error is kept for responses with explicit response options. It also passes
// through registered renderers, preserving existing call sites while allowing
// central error classification.
func Error(err error, options ...responseOption) Response {
	return renderError(exception.New(err), options)
}

func renderError(input exception.Exception, options []responseOption) Response {
	for _, renderer := range exceptionRenderers {
		rendered := renderer(input)
		if rendered == nil {
			continue
		}

		if renderedOptions, ok := rendered.(*responseOptions); ok {
			for _, option := range options {
				option(renderedOptions)
			}
		}

		return rendered
	}

	defaultOptions := []responseOption{
		WithError(input.Err),
		WithStatusCode(http.StatusInternalServerError),
	}

	return JSON(append(defaultOptions, options...)...)
}

func BadRequest(options ...responseOption) Response {
	return JSON(append(options, WithStatusCode(http.StatusBadRequest))...)
}

func Unprocessable(options ...responseOption) Response {
	return JSON(append(options, WithStatusCode(http.StatusUnprocessableEntity))...)
}

func Unauthenticated(options ...responseOption) Response {
	defaultOptions := []responseOption{
		WithMessage("You are not authenticated"),
		WithStatusCode(http.StatusUnauthorized),
	}

	return JSON(append(defaultOptions, options...)...)
}

func Forbidden(message string, options ...responseOption) Response {
	defaultOptions := []responseOption{
		WithMessage(message),
		WithStatusCode(http.StatusForbidden),
	}

	return JSON(append(defaultOptions, options...)...)
}

func NotFound(options ...responseOption) Response {
	return JSON(append(options, WithStatusCode(http.StatusNotFound))...)
}

func Conflict(options ...responseOption) Response {
	return JSON(append(options, WithStatusCode(http.StatusConflict))...)
}

func Timeout(options ...responseOption) Response {
	return JSON(append(options, WithStatusCode(http.StatusGatewayTimeout))...)
}

func TooManyRequests(options ...responseOption) Response {
	return JSON(append(options, WithStatusCode(http.StatusTooManyRequests))...)
}
