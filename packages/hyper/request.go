package hyper

import (
	"context"
	"fmt"
	"github.com/gonstruct/core/otel"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type requestOption func(*Request)

type RequestOption = requestOption

func NewRequest(options ...requestOption) *Request {
	return new(Request).apply(options...)
}

type Request struct {
	ctx     context.Context
	timeout time.Duration
	method  string
	base    string
	url     string

	headers     *headers
	queryParams *queryParams
	body        body
}

func (request *Request) apply(options ...requestOption) *Request {
	for _, option := range options {
		option(request)
	}
	return request
}

func (request *Request) makeUrl(path string) string {
	u, _ := url.Parse(path)

	if request.base != "" {
		u, _ = url.Parse(request.base)
		u = u.JoinPath(path)
	}

	if request.queryParams != nil {
		u.RawQuery = request.queryParams.Encode()
	}

	return u.String()
}

// otelClient returns an HTTP client instrumented with OpenTelemetry.
var otelClient = &http.Client{
	Transport: otelhttp.NewTransport(otel.NewTransport(nil)),
}

func (request *Request) do() *Response {
	var body io.Reader
	if request.body != nil {
		body = request.body.Read()
	}

	// Use context if provided, otherwise use background context
	ctx := request.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	if request.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.timeout)
		defer cancel()
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, request.method, request.makeUrl(request.url), body)
	if err != nil {
		return request.respond(nil, request.error(err, "failed to create request"))
	}

	if request.headers != nil {
		req.Header = request.headers.Header
	}

	res, err := otelClient.Do(req)
	// NOT A FAN OF THIS:
	if err != nil || res == nil {
		fmt.Printf("[hyper] %s %s -> error: %v (%s)\n", request.method, req.URL.String(), err, time.Since(start))
	} else {
		fmt.Printf("[hyper] %s %s -> %d (%s)\n", request.method, req.URL.String(), res.StatusCode, time.Since(start))
	}
	return request.respond(res, err)
}

func (request *Request) respond(res *http.Response, err error) *Response {
	var body []byte

	if res != nil && res.Body != nil {
		defer res.Body.Close()
		body, err = io.ReadAll(res.Body)
	}

	return &Response{
		Response: res,
		request:  request,
		err:      err,
		body:     body,
	}
}

func (r *Request) error(err error, msg ...string) error {
	var prefix string
	if len(msg) == 1 {
		prefix = msg[0] + " "
	}

	return fmt.Errorf("[hyper] %s(method: %s, url: %s): %w", prefix, r.method, r.makeUrl(r.url), err)
}
