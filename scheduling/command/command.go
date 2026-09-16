package command

import "context"

type Command interface {
	Name() string
	Handle(context.Context) error
}
