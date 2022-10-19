package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShouldFailIfNotExistingFile(t *testing.T) {
	_, err := NewConfig("badpath")
	require.Error(t, err)
}

func TestShouldFailIfConfigIsInvalidYaml(t *testing.T) {
	_, err := NewConfig("./testdata/invalid_config.env")
	require.Error(t, err)
}

func TestGetEnvShouldReturnDefaultIfKeyNotFound(t *testing.T) {
	require.Equal(t, "def", getEnv("aaaaa", "def"))
}

func TestGetEnvAsIntShouldReturnDefaultIfKeyNotFound(t *testing.T) {
	require.Equal(t, 1337, getEnvAsInt("aaaaa", 1337))
}

func TestGetEnvAsDurationShouldReturnDefaultIfKeyNotFound(t *testing.T) {
	require.Equal(t, time.Second*1337, getEnvAsDuration("aaaaa", time.Second*1337))
}

func TestGetEnvAsDurationShouldReturnDefaultIfKeyHasInvalidValue(t *testing.T) {
	require.NoError(t, os.Setenv("testaaa", "a"))
	require.Equal(t, time.Second*1337, getEnvAsDuration("testaaa", time.Second*1337))
}
