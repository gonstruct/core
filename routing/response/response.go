package response

import (
	"github.com/gin-gonic/gin"
)

type Response interface {
	resolve(c *gin.Context)
}

type (
	Handler      func(context *gin.Context) Response
	HandlerChain []Handler
)

func (chain HandlerChain) Handle() gin.HandlersChain {
	return HandleChain(chain)
}

func HandleChain(handlers HandlerChain) gin.HandlersChain {
	handlersChain := make(gin.HandlersChain, len(handlers))
	for i, handler := range handlers {
		handlersChain[i] = Handle(handler)
	}

	return handlersChain
}

func Handle(handler Handler) gin.HandlerFunc {
	return func(context *gin.Context) { handler(context).resolve(context) }
}
