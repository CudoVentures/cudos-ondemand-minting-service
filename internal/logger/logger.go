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

func (l *logger) Info(msg string) {
	l.zlogger.Info().Msg(msg)
}

func (l *logger) Infof(format string, v ...interface{}) {
	l.zlogger.Info().Msgf(format, v...)
}

func (l *logger) Warn(msg string) {
	l.zlogger.Warn().Msg(msg)
}

func (l *logger) Warnf(format string, v ...interface{}) {
	l.zlogger.Warn().Msgf(format, v...)
}

type logger struct {
	zlogger zerolog.Logger
}
