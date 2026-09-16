package response

import "github.com/gin-gonic/gin"

type responseNext struct {
	after func()
}

func (r *responseNext) resolve(c *gin.Context) {
	c.Next()

	if r.after != nil {
		r.after()
	}
}

func Next(after ...func()) Response {
	response := &responseNext{}
	if len(after) == 1 {
		response.after = after[0]
	}
	return response
}

type responseNil struct{}

func (r *responseNil) resolve(c *gin.Context) {}

func Nil() Response {
	return new(responseNil)
}
