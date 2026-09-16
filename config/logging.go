package config

type LogChannel string

const (
	ConsoleLogChannel LogChannel = "console"
	JSONLogChannel    LogChannel = "json"
	StackLogChannel   LogChannel = "stack"
)

func (LogChannel) Values() []LogChannel {
	return []LogChannel{
		ConsoleLogChannel,
		JSONLogChannel,
		StackLogChannel,
	}
}

func (channel LogChannel) String() string {
	return string(channel)
}

type Logging struct {
	Channel LogChannel
}
