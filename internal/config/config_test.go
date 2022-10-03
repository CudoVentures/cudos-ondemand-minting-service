package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldFailIfNotExistingFile(t *testing.T) {
	_, err := NewConfig("badpath")
	require.Error(t, err)
}

func TestShouldFailIfConfigIsInvalidYaml(t *testing.T) {
	_, err := NewConfig("./testdata/invalid_config.yaml")
	require.Error(t, err)
}
