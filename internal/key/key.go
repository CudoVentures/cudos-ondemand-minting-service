package key

import (
	cryptohd "github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/go-bip39"
)

func PrivKeyFromMnemonic(mnemonic string) (*secp256k1.PrivKey, error) {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	masterPriv, ch := cryptohd.ComputeMastersFromSeed(seed)

	derivedKey, err := cryptohd.DerivePrivateKeyForPath(masterPriv, ch, hdPath)
	if err != nil {
		return nil, err
	}

	return &secp256k1.PrivKey{Key: derivedKey}, nil
}

var hdPath = "m/44'/118'/0'/0/0"
