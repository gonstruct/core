package hyper

import (
	"context"
	"fmt"
	"net/http"
)

func main() {
	// hyper.Get("/test", hyper.WithHeaders(hyper.H{"X-Test": "test"})

	// hyper.Get("/test",
	// 	hyper.WithHeaders(
	// 		hyper.Header("X-Test", "test"),
	// 		hyper.Header("X-Test2", "test2"),
	// 	),
	// 	hyper.WithQuery(
	// 		hyper.Query("param1", "value1"),
	// 		hyper.Query("param2", "value2"),
	// 	),
	// )

	res := Get("/test")
	if !res.Ok() {
		panic(res)
	}
	if res.Ok() {
		value, err := Json[int](res)
		if err != nil {
			panic(err)
		}
		fmt.Println("ok", value)
	}

	res = NewRequest().Get("/test")
	if !res.Ok() {
		panic(res)
	}

	// NewRequest()

	// hyper.Base(
	// 	hyper.WithHeaders(hyper.H{
	// 		"X-Test": "test",
	// 	}),
	// ).Post("/test/{id}")

	// Get(hyper.Url("https://google.com/profile", 133, "hello world"))
}

func (request *Request) Get(url string, options ...requestOption) *Response {
	request.method = http.MethodGet
	request.url = url
	return request.apply(options...).do()
}

func Get(url string, options ...requestOption) *Response {
	return NewRequest(options...).Get(url)
}

func (request *Request) GetCtx(context context.Context, url string, options ...requestOption) *Response {
	request.method = http.MethodGet
	request.ctx = context
	request.url = url
	return request.apply(options...).do()
}

func GetCtx(context context.Context, url string, options ...requestOption) *Response {
	return NewRequest(options...).GetCtx(context, url)
}

func (request *Request) Post(url string, options ...requestOption) *Response {
	request.method = http.MethodPost
	request.url = url
	return request.apply(options...).do()
}

func (request *Request) PostCtx(context context.Context, url string, options ...requestOption) *Response {
	request.method = http.MethodPost
	request.ctx = context
	request.url = url
	return request.apply(options...).do()
}

func Post(url string, options ...requestOption) *Response {
	return NewRequest(options...).Post(url)
}

func PostCtx(context context.Context, url string, options ...requestOption) *Response {
	return NewRequest(options...).PostCtx(context, url)
}

func (request *Request) Delete(url string, options ...requestOption) *Response {
	request.method = http.MethodDelete
	request.url = url
	return request.apply(options...).do()
}

func Delete(url string, options ...requestOption) *Response {
	return NewRequest(options...).Delete(url)
}

func (request *Request) DeleteCtx(context context.Context, url string, options ...requestOption) *Response {
	request.method = http.MethodDelete
	request.ctx = context
	request.url = url
	return request.apply(options...).do()
}

func DeleteCtx(context context.Context, url string, options ...requestOption) *Response {
	return NewRequest(options...).DeleteCtx(context, url)
}
