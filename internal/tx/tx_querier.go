package tx

import (
	"context"
	"fmt"
	"time"

	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func NewTxQuerier(node txSearcher) *txQuerier {
	return &txQuerier{node: node}
}

func (tq *txQuerier) Query(ctx context.Context, query string) (*ctypes.ResultTxSearch, error) {
	// TODO: Do we need to do pagination? Will we get all results if we don't use pagination or there is some limit?
	results, err := tq.node.TxSearch(ctx, query, true, nil, nil, "asc")
	if err != nil {
		return nil, fmt.Errorf("tx search (%s) failed: %s", query, err)
	}

	return results, nil
}

const txSearchTimeout = 10 * time.Second

type txQuerier struct {
	node txSearcher
}

type txSearcher interface {
	TxSearch(ctx context.Context, query string, prove bool, page, perPage *int, orderBy string) (*ctypes.ResultTxSearch, error)
}
