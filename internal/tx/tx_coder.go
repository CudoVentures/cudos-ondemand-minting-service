package tx

import (
	"github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tmtypes "github.com/tendermint/tendermint/types"
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
