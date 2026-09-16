package response

import "github.com/gin-gonic/gin"

type jsonBodyResponse struct {
	statusCode int
	body       any
	err        error
}

func (self *jsonBodyResponse) resolve(context *gin.Context) {
	if self.err != nil {
		_ = context.Error(self.err)
	}

	if self.statusCode >= 400 {
		context.AbortWithStatusJSON(self.statusCode, self.body)
		return
	}

	context.JSON(self.statusCode, self.body)
}

func JSONBody(statusCode int, body any, errors ...error) Response {
	response := &jsonBodyResponse{statusCode: statusCode, body: body}
	if len(errors) > 0 {
		response.err = errors[0]
	}

	return response
}
