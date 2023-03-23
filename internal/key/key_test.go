package key

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestShouldGetPrivKeyFromMnemonic(t *testing.T) {
	privKey, err := PrivKeyFromMnemonic("rebel wet poet torch carpet gaze axis ribbon approve depend inflict menu")
	require.NotNil(t, privKey)
	require.NoError(t, err)
	require.Equal(t, "cosmos1a326k254fukx9jlp0h3fwcr2ymjgludza2npne", sdk.AccAddress(privKey.PubKey().Address()).String())
}

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
