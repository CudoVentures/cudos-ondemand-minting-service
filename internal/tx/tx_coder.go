package tx

import (
	"cosmossdk.io/simapp/params"
	tmtypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewTxCoder(encodingConfig *params.EncodingConfig) *txCoder {
	return &txCoder{encodingConfig: encodingConfig}
}

func (tc *txCoder) Decode(tx tmtypes.Tx) (sdk.Tx, error) {
	return tc.encodingConfig.TxConfig.TxDecoder()(tx)
}

type txCoder struct {
	encodingConfig *params.EncodingConfig
}
