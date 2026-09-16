package response

import (
	"fmt"
	"github.com/gonstruct/core/routing/resource"
	"net/http"
	"os"
	"sort"

	"github.com/gin-gonic/gin"
)

type ErrorsMap map[string]string

type value[T any] struct {
	isSet bool
	value T
}

func newValue[T any](input T) value[T] {
	return value[T]{isSet: true, value: input}
}

func (v value[T]) Value(fallback T) T {
	if v.isSet {
		return v.value
	}
	return fallback
}

type responseOptions struct {
	StatusCode int
	Message    value[string]
	Code       value[string]
	Data       value[any]

	Resource           value[resource.V2ResourceInterface]
	Resources          value[[]resource.Resource] // NOTe: could be done better
	ResourceCollection value[resource.V2CollectionInterface]

	Error  value[error]
	Errors value[ErrorsMap]
	Meta   value[meta]

	Raw   map[string]any
	After func()
}

func (options *responseOptions) resolve(context *gin.Context) {
	if options.StatusCode == http.StatusNoContent || options.StatusCode == http.StatusNotModified {
		context.JSON(options.StatusCode, nil)

		if options.After != nil {
			options.After()
		}
		return
	}

	if options.Resource.isSet {
		attributes := options.Resource.value.Attributes(context)
		if options.Resources.isSet {
			attributes = attributes.Merge(options.Resources.value...)
		}

		options.Data = newValue[any](attributes)
	}

	if options.ResourceCollection.isSet {
		options.Data = newValue[any](options.ResourceCollection.value.Attributes(context))
	}

	if options.StatusCode >= 400 {
		// A 4xx is an answer, not a fault: the client asked for something it
		// may not have. Only a 5xx is ours.
		//
		// gin's error channel is what otelgin reads to mark the span failed,
		// and a failed span is force-kept by the collector's error sampling
		// policy — so recording a 401 here would both fill the error list with
		// logged-out visitors and spend the trace budget on them. 4xx traces
		// are still kept, by the status-code policy, and bodies still captured.
		if options.Error.isSet && options.StatusCode >= 500 {
			context.Error(options.Error.value)
		}

		context.AbortWithStatusJSON(options.StatusCode, resource.Resource{
			"message": resource.When(options.Message.isSet, func() string { return options.Message.value }),
			"code":    resource.When(options.Code.isSet, func() string { return options.Code.value }),
			"error":   resource.When(options.Error.isSet && os.Getenv("APP_ENV") != "production", func() string { return options.Error.value.Error() }),
			"errors":  resource.When(options.Errors.isSet, func() ErrorsMap { return options.Errors.value }),
		})

		if options.After != nil {
			options.After()
		}
		return
	}

	context.JSON(options.StatusCode, resource.Resource{
		"data":    resource.When(options.Data.isSet, func() any { return options.Data.value }),
		"meta":    resource.When(options.Meta.isSet, func() any { return options.Meta.value.Resource() }),
		"message": resource.When(options.Message.isSet, func() string { return options.Message.value }),
	}.Merge(options.Raw))

	if options.After != nil {
		options.After()
	}
}

func JSON(options ...responseOption) Response {
	opts := &responseOptions{}

	for _, option := range options {
		option(opts)
	}

	return opts
}

type responseOption func(*responseOptions)

func WithStatusCode(statusCode int) responseOption {
	return func(options *responseOptions) {
		options.StatusCode = statusCode
	}
}

func WithMessageF(message string, args ...any) responseOption {
	return func(options *responseOptions) {
		options.Message = newValue(fmt.Sprintf(message, args...))
	}
}

func WithMessage(message string) responseOption {
	return func(options *responseOptions) {
		options.Message = newValue(message)
	}
}

func WithCode(code string) responseOption {
	return func(options *responseOptions) {
		options.Code = newValue(code)
	}
}

func WithData(data any) responseOption {
	return func(options *responseOptions) {
		options.Data = newValue(data)
	}
}

func WithRaw(raw map[string]any) responseOption {
	return func(options *responseOptions) {
		options.Raw = raw
	}
}

func WithResource(resource resource.ResourceInterface, other ...resource.Resource) responseOption {
	return func(options *responseOptions) {
		data := resource.Attributes()
		for _, r := range other {
			data = data.Merge(r)
		}

		options.Data = newValue[any](data)
	}
}

func WithResourceCollection(collection resource.Collection) responseOption {
	return func(options *responseOptions) {
		options.Data = newValue[any](collection)
	}
}

func WithResourceV2(resource resource.V2ResourceInterface, other ...resource.Resource) responseOption {
	return func(options *responseOptions) {
		options.Resource = newValue(resource)

		if len(other) > 0 {
			options.Resources = newValue(other)
		}
	}
}

func WithResourceV2Collection(collection resource.V2CollectionInterface) responseOption {
	return func(options *responseOptions) {
		options.ResourceCollection = newValue(collection)
	}
}

func WithError(err error) responseOption {
	return func(options *responseOptions) {
		options.Error = newValue(err)
	}
}

func WithErrors(errors map[string]string) responseOption {
	return func(options *responseOptions) {
		options.Errors = newValue[ErrorsMap](errors)

		if options.Message.isSet || len(errors) == 0 {
			return
		}

		fields := make([]string, 0, len(errors))
		for field := range errors {
			fields = append(fields, field)
		}
		sort.Strings(fields)

		message := errors[fields[0]]
		if len(fields) > 1 {
			message += fmt.Sprintf(" and %d more errors", len(fields)-1)
		}

		options.Message = newValue(message)
	}
}

func WithOffsetPagination(count int64, query offsetPaginationQuery) responseOption {
	return func(options *responseOptions) {
		var length int
		if options.ResourceCollection.isSet {
			length = options.ResourceCollection.value.Length()
		} else {
			collection, ok := options.Data.value.(resource.Collection)
			if !ok {
				panic("WithOffsetPagination requires Data to be a resource.Collection")
			}
			length = len(collection)
		}

		options.Meta = newValue[meta](makeOffsetPaginationMeta(count, length, query))
	}
}

func WithAfter(fn func()) responseOption {
	return func(options *responseOptions) {
		options.After = fn
	}
}
