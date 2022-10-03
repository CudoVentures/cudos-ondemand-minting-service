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
	require.Equal(t, "<nil> ERR  error=test\n", buf.String())
}
