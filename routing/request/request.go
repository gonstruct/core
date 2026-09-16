package request

import (
	"github.com/gonstruct/core/routing/request/validation"
	"github.com/gonstruct/core/routing/response"

	"github.com/gin-gonic/gin"
)

type Request[P, Q, B any] struct {
	*Context

	Params    *P
	Query     *Q
	Validated *B
}

type requestInterface[P, Q, B any] interface {
	Authorize(*Context) response.Response
	Params(*Context) validation.Result[P]
	Query(*Context) validation.Result[Q]
	Json(*Context) validation.Result[B]
}

type requestWithPrepareForValidation interface {
	PrepareForValidation(request *Context) error
}

type requestWithPassedValidation[P, Q, B any] interface {
	PassedValidation(request *Request[P, Q, B]) response.Response
}

func New[R requestInterface[P, Q, B], P, Q, B any](context *gin.Context) (request *Request[P, Q, B], err response.Response) {
	request = &Request[P, Q, B]{
		Context: NewContext(context),
	}

	var requester R
	if response := requester.Authorize(request.Context); response != nil {
		return nil, response
	}

	if prepareRequest, ok := any(requester).(requestWithPrepareForValidation); ok {
		if err := prepareRequest.PrepareForValidation(request.Context); err != nil {
			return nil, response.Error(err, response.WithMessage("Failed to prepare request for validation"))
		}
	}

	jsonFunc := requester.Json(request.Context)
	if jsonFunc != nil {
		validated, err := jsonFunc("json")
		if err != nil {
			return nil, err
		}
		if any(validated) != nil {
			request.Validated = &validated
		}
	}

	paramsFunc := requester.Params(request.Context)
	if paramsFunc != nil {
		params, err := paramsFunc("params")
		if err != nil {
			return nil, err
		}

		if any(params) != nil {
			request.Params = &params
		}
	}

	queryFunc := requester.Query(request.Context)
	if queryFunc != nil {
		query, err := queryFunc("query")
		if err != nil {
			return nil, err
		}

		if any(query) != nil {
			request.Query = &query
		}
	}

	if passedValidation, ok := any(requester).(requestWithPassedValidation[P, Q, B]); ok {
		if validationResponse := passedValidation.PassedValidation(request); validationResponse != nil {
			return nil, validationResponse
		}
	}

	return request, nil
}
