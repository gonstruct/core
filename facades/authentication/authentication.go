package authentication

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Session interface {
	ID() string
	Lifetime() time.Duration
	Set(context *gin.Context)
}

type Token interface {
	ID() string
	String() string
	ExpiresAt() time.Time
}

type service interface {
	GenerateToken(arg Session) (Token, error)
	ValidateToken(token string, strict ...bool) (Session, error)
}

var instance service

func Register(service service) {
	instance = service
}

func GenerateToken(arg Session) (Token, error) {
	return instance.GenerateToken(arg)
}

func ValidateToken(token string, strict ...bool) (Session, error) {
	return instance.ValidateToken(token, strict...)
}
