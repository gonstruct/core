package console

import "fmt"

type Handler func(console *Console, arguments []string) error

type Command struct {
	Name   string
	Handle Handler
}

type Console struct {
	commands map[string]Handler
}

func New() *Console {
	return &Console{commands: map[string]Handler{}}
}

func (self *Console) Register(command string, handler Handler) {
	self.commands[command] = handler
}

func (self *Console) Add(commands ...Command) {
	for _, command := range commands {
		self.commands[command.Name] = command.Handle
	}
}

func (self *Console) Run(command string, arguments []string) error {
	handler, exists := self.commands[command]
	if !exists {
		return fmt.Errorf("unknown command: %s", command)
	}

	return handler(self, arguments)
}
