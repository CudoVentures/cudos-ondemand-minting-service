package logger

import (
	"github.com/rs/zerolog"
)

func NewLogger(zlogger zerolog.Logger) *logger {
	return &logger{zlogger: zlogger}
}

func (l *logger) Error(err error) {
	l.zlogger.Error().Err(err).Send()
}

type logger struct {
	zlogger zerolog.Logger
}
