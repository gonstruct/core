package response

import "github.com/gin-gonic/gin"

type responseRedirect struct {
	StatusCode int
	Location   string
}

func (r *responseRedirect) resolve(c *gin.Context) {
	c.Redirect(r.StatusCode, r.Location)
}

func Redirect(statusCode int, location string) Response {
	return &responseRedirect{StatusCode: statusCode, Location: location}
}
