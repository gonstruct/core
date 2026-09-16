package hyper

import (
	"bytes"
	"encoding/json"
	"io"
)

type body interface {
	Read() io.Reader
}

func newJsonBody() *bodyJson {
	return &bodyJson{data: map[string]any{}}
}

type bodyJson struct {
	data map[string]any
}

type (
	jsonFieldOption func(*bodyJson)
	J               map[string]any
)

func (body *bodyJson) Read() io.Reader {
	data, err := json.Marshal(body.data)
	if err != nil {
		return nil
	}

	return bytes.NewReader(data)
}

func newBytesBody(data []byte) *bodyBytes {
	return &bodyBytes{data: data}
}

type bodyBytes struct {
	data []byte
}

func (body *bodyBytes) Read() io.Reader {
	return bytes.NewReader(body.data)
}

func WithJsonField(key string, value any) jsonFieldOption {
	return func(b *bodyJson) {
		b.data[key] = value
	}
}

func (j J) toJsonFieldOptions() []jsonFieldOption {
	options := make([]jsonFieldOption, 0, len(j))
	for key, value := range j {
		options = append(options, WithJsonField(key, value))
	}
	return options
}

func WithJson[M J | jsonFieldOption](mapOrOptions ...M) requestOption {
	return func(request *Request) {
		WithHeaders(WithHeader("Content-Type", "application/json"))(request)

		if request.body == nil {
			request.body = newJsonBody()
		} else if _, ok := request.body.(*bodyJson); !ok {
			panic("[hyper] cannot use WithJson with non-json body")
		}

		for _, item := range mapOrOptions {
			switch v := any(item).(type) {
			case J:
				for _, option := range v.toJsonFieldOptions() {
					option(request.body.(*bodyJson))
				}
			case jsonFieldOption:
				v(request.body.(*bodyJson))
			}
		}
	}
}

func WithJsonStruct(value any) requestOption {
	return func(request *Request) {
		WithHeaders(WithHeader("Content-Type", "application/json"))(request)

		if request.body != nil {
			switch request.body.(type) {
			case *bodyJson, *bodyBytes:
			default:
				panic("[hyper] cannot use WithJsonStruct with non-json body")
			}
		}

		data, err := json.Marshal(value)
		if err != nil {
			panic("[hyper] failed to marshal json struct: " + err.Error())
		}

		request.body = newBytesBody(data)
	}
}
