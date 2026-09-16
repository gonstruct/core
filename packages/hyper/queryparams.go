package hyper

import (
	"fmt"
	"net/url"
)

func newQueryParams() *queryParams {
	return &queryParams{url.Values{}}
}

type queryParams struct {
	url.Values
}

type (
	queryParamOption func(*queryParams)
	Q                map[string]any
)

func WithQuery(key string, value ...any) queryParamOption {
	return func(q *queryParams) {
		for _, v := range value {
			q.Add(key, fmt.Sprint(v))
		}
	}
}

func (q Q) toQueryParamOptions() []queryParamOption {
	options := make([]queryParamOption, 0, len(q))
	for key, value := range q {
		options = append(options, WithQuery(key, value))
	}
	return options
}

func WithQueries[M Q | queryParamOption](mapOrOptions ...M) requestOption {
	return func(request *Request) {
		if request.queryParams == nil {
			request.queryParams = newQueryParams()
		}

		for _, item := range mapOrOptions {
			switch v := any(item).(type) {
			case Q:
				for _, option := range v.toQueryParamOptions() {
					option(request.queryParams)
				}
			case queryParamOption:
				v(request.queryParams)
			}
		}
	}
}
