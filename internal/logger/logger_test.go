package logger

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestShouldNotBlock(t *testing.T) {
	zlogger := NewLogger(zerolog.New(os.Stderr).With().Timestamp().Logger())
	zlogger.Error(errors.New("test"))
}

func TestShouldLogSuccessfully(t *testing.T) {
	buf := &bytes.Buffer{}
	log := zerolog.New(zerolog.ConsoleWriter{Out: buf, NoColor: true})
	zlogger := NewLogger(log.With().Logger())
	zlogger.Error(errors.New("test"))
	zlogger.Info("test")
	zlogger.Infof("test%s", "test")
	zlogger.Warn("test")
	zlogger.Warnf("test%s", "test")
	require.Equal(t, "<nil> ERR  error=test\n<nil> INF test\n<nil> INF testtest\n<nil> WRN test\n<nil> WRN testtest\n", buf.String())
}
