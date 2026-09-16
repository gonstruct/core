// Package httpcapture buffers request and response bodies so they can be
// attached to a span after the outcome of the request is known.
package httpcapture

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"
)

// Request wraps a request body and captures bytes as downstream handlers read
// them. Nothing is read up front, so a request whose body is never consumed
// costs nothing.
type Request struct {
	original io.ReadCloser
	buffer   bytes.Buffer
	limit    int
	capped   bool
}

// NewRequest replaces the body on context with a capturing wrapper. A body that
// is nil, or whose declared Content-Length already exceeds limit, is left alone
// and Body returns nil.
func NewRequest(context *gin.Context, limit int) *Request {
	request := &Request{limit: limit}

	if context.Request.Body == nil || context.Request.ContentLength > int64(limit) {
		return request
	}

	request.original = context.Request.Body
	context.Request.Body = request

	return request
}

func (self *Request) Read(data []byte) (int, error) {
	if self.original == nil {
		return 0, io.EOF
	}

	read, err := self.original.Read(data)
	if read > 0 && !self.capped {
		self.capture(data[:read])
	}

	return read, err
}

func (self *Request) Close() error {
	if self.original == nil {
		return nil
	}

	return self.original.Close()
}

// Body returns the bytes captured so far, or nil when nothing was captured.
func (self *Request) Body() []byte {
	if self.buffer.Len() == 0 {
		return nil
	}

	return self.buffer.Bytes()
}

// capture appends up to limit bytes. bytes.Buffer.Write always returns a nil
// error (it panics on OOM), so the return values are intentionally discarded.
func (self *Request) capture(data []byte) {
	remaining := self.limit - self.buffer.Len()
	if remaining <= 0 {
		self.capped = true

		return
	}

	if len(data) > remaining {
		_, _ = self.buffer.Write(data[:remaining])
		self.capped = true

		return
	}

	_, _ = self.buffer.Write(data)
}
