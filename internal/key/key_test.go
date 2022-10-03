package key

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldFailIfInvalidMnemonic(t *testing.T) {
	privKey, err := PrivKeyFromMnemonic("bad mnemonic")
	require.Nil(t, privKey)
	require.Error(t, err)
}

func TestShouldFailIfInvalidHdPath(t *testing.T) {
	hdPath = "badpath"
	privKey, err := PrivKeyFromMnemonic("rebel wet poet torch carpet gaze axis ribbon approve depend inflict menu")
	require.Nil(t, privKey)
	require.Error(t, err)
}
