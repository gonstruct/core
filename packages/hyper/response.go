package hyper

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tidwall/gjson"
)

type Response struct {
	*http.Response

	request *Request
	err     error
	body    []byte
}

// Body is the raw bytes, for callers that need a provider's error text.
func (response *Response) Body() []byte {
	return response.body
}

func (response *Response) Error() error {
	return response.err
}

func (response *Response) Ok() bool {
	if response.err != nil {
		return false
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return true
	}

	response.err = response.request.error(fmt.Errorf("request failed with status %d", response.StatusCode), "request failed")
	return false
}

func (response *Response) Json(value any) error {
	return json.Unmarshal(response.body, value)
}

func Json[R any](res *Response, keyPath ...string) (*R, error) {
	if len(keyPath) == 0 {
		var result R
		if err := res.Json(&result); err != nil {
			fmt.Println("[hyper/response] failed to unmarshal json, raw body:", string(res.body))
			return nil, err
		}

		return &result, nil
	}

	val := gjson.GetBytes(res.body, keyPath[0])
	if !val.Exists() {
		return nil, res.request.error(fmt.Errorf("key %s not present in response", keyPath[0]))
	}

	var result R
	if err := json.Unmarshal([]byte(val.Raw), &result); err != nil {
		return nil, res.request.error(err, "failed to unmarshal json")
	}

	return &result, nil
}
