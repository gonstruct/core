package response

import (
	"net/http"
)

func Success(options ...responseOption) Response {
	return JSON(append(options, WithStatusCode(http.StatusOK))...)
}

func Created(options ...responseOption) Response {
	return JSON(append(options, WithStatusCode(http.StatusCreated))...)
}

func NoContent() Response {
	return JSON(WithStatusCode(http.StatusNoContent))
}
