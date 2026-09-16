package request

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewContext(context *gin.Context) *Context {
	return &Context{Context: context}
}

type Context struct {
	*gin.Context
}

func (c *Context) GetRequest() *http.Request {
	return c.Context.Request
}
