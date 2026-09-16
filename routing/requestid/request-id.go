package requestid

import (
	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
)

const (
	HeaderName = "X-Request-ID"
	contextKey = "request_id"
)

func New() string {
	return "req_" + ulid.Make().String()
}

func Set(context *gin.Context, requestID string) {
	context.Set(contextKey, requestID)
	context.Header(HeaderName, requestID)
}

func Get(context *gin.Context) string {
	requestID, _ := context.Get(contextKey)
	value, _ := requestID.(string)
	return value
}

func Ensure(context *gin.Context) string {
	if requestID := Get(context); requestID != "" {
		return requestID
	}

	requestID := New()
	Set(context, requestID)
	return requestID
}
