package tx

import (
	"cosmossdk.io/simapp/params"
	client "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	auth "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

func NewTxSigner(encodingConfig *params.EncodingConfig, privKey *secp256k1.PrivKey) *txSigner {
	return &txSigner{
		encodingConfig: encodingConfig,
		privKey:        privKey,
	}
}

func (ts *txSigner) SetMsgs(tx client.TxBuilder, msgs ...sdk.Msg) error {
	return tx.SetMsgs(msgs...)
}

func (ts *txSigner) SetSignatures(tx client.TxBuilder, signatures ...signingtypes.SignatureV2) error {
	return tx.SetSignatures(signatures...)
}

func (ts *txSigner) GetSignBytes(mode signing.SignMode, data auth.SignerData, tx sdk.Tx) ([]byte, error) {
	return ts.encodingConfig.TxConfig.SignModeHandler().GetSignBytes(mode, data, tx)
}

func (ts *txSigner) Sign(msg []byte) ([]byte, error) {
	return ts.privKey.Sign(msg)
}

type txSigner struct {
	encodingConfig *params.EncodingConfig
	privKey        *secp256k1.PrivKey
}
