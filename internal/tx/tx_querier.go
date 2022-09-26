package tx

import (
	"context"
	"fmt"
	"time"

	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func NewTxQuerier(node *rpchttp.HTTP) *txQuerier {
	return &txQuerier{node: node}
}

func (tq *txQuerier) Query(query string) (*ctypes.ResultTxSearch, error) {
	txSearchCtx, cancelFunc := context.WithTimeout(context.Background(), txSearchTimeout)
	defer cancelFunc()

	// TODO: Do we need to do pagination? Will we get all results if we don't use pagination or there is some limit?
	results, err := tq.node.TxSearch(txSearchCtx, query, true, nil, nil, "asc")
	if err != nil {
		return nil, fmt.Errorf("tx search (%s) failed: %s", query, err)
	}

	return results, nil
}

const txSearchTimeout = 10 * time.Second

type txQuerier struct {
	node *rpchttp.HTTP
}
