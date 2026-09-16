package request

import (
	"encoding/json"
	"errors"
)

func (context *Context) BindJson(obj any) error {
	if context.Request == nil || context.Request.Body == nil {
		return errors.New("No request body provided")
	}

	return json.NewDecoder(context.Request.Body).Decode(obj)
}

func (c *Context) BindUri(obj any) error {
	return c.ShouldBindUri(obj)
}

func (c *Context) BindQuery(obj any) error {
	return c.ShouldBindQuery(obj)
}
