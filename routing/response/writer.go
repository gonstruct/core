package response

import "github.com/gin-gonic/gin"

type writerResponse struct{ write func(*gin.Context) }

func (response *writerResponse) resolve(context *gin.Context) { response.write(context) }

// Writer is for non-JSON responses such as the media proxy. The callback owns
// the status, headers, and body.
func Writer(write func(*gin.Context)) Response { return &writerResponse{write: write} }
