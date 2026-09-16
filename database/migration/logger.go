package migration

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

type logger struct{}

func (logger) Printf(format string, v ...any) {
	log.Info().Msg(message(format, v...))
}

func (logger) Fatalf(format string, v ...any) {
	log.Fatal().Msg(message(format, v...))
}

func message(format string, v ...any) string {
	return strings.TrimPrefix(strings.TrimSpace(fmt.Sprintf(format, v...)), "goose: ")
}
