package httpcapture

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response captures the response body as it is written.
//
// Only failed responses are ever read back, so successful ones are not buffered
// at all. Gin sets the status before the first write, which is what makes that
// check possible here.
type Response struct {
	gin.ResponseWriter

	buffer bytes.Buffer
	limit  int
}

func NewResponse(writer gin.ResponseWriter, limit int) *Response {
	return &Response{ResponseWriter: writer, limit: limit}
}

func (self *Response) Write(data []byte) (int, error) {
	self.capture(data)

	return self.ResponseWriter.Write(data)
}

func (self *Response) WriteString(data string) (int, error) {
	self.capture([]byte(data))

	return self.ResponseWriter.WriteString(data)
}

// Body returns the captured response body bytes.
func (self *Response) Body() []byte {
	return self.buffer.Bytes()
}

// capture appends up to limit bytes. bytes.Buffer.Write always returns a nil
// error (it panics on OOM), so the return values are intentionally discarded.
func (self *Response) capture(data []byte) {
	if self.Status() < http.StatusBadRequest {
		return
	}

	remaining := self.limit - self.buffer.Len()
	if remaining <= 0 {
		return
	}

	if len(data) > remaining {
		_, _ = self.buffer.Write(data[:remaining])

		return
	}

	_, _ = self.buffer.Write(data)
}
