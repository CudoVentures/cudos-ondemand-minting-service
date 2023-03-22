package tx

import (
	"context"
	"fmt"

	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func NewTxQuerier(node txSearcher) *txQuerier {
	return &txQuerier{node: node}
}

func (tq *txQuerier) Query(ctx context.Context, query string) (*ctypes.ResultTxSearch, error) {
	var allResults *ctypes.ResultTxSearch = nil

	for page := 1; ; page += 1 {
		// fmt.Printf("Fetching page %d\n", page)
		results, err := tq.node.TxSearch(ctx, query, true, &page, &itemsPerPage, "asc")
		if err != nil {
			return nil, fmt.Errorf("tx search (%s) failed: %s", query, err)
			// if allResults == nil {
			// 	return nil, fmt.Errorf("tx search (%s) failed: %s", query, err)
			// } else {
			// 	fmt.Println(err)
			// 	continue
			// }
		}

		if allResults == nil {
			allResults = results
		} else {
			allResults.Txs = append(allResults.Txs, results.Txs...)
			allResults.TotalCount = results.TotalCount
		}

		from := (page - 1) * itemsPerPage
		to := page*itemsPerPage - 1
		// fmt.Printf("[%d; %d] -> %d\n", from, to, allResults.TotalCount)
		if from <= allResults.TotalCount && allResults.TotalCount <= to {
			break
		}
	}

	return allResults, nil
}

// const txSearchTimeout = 10 * time.Second

var itemsPerPage = 100

type txQuerier struct {
	node txSearcher
}

type txSearcher interface {
	TxSearch(ctx context.Context, query string, prove bool, page, perPage *int, orderBy string) (*ctypes.ResultTxSearch, error)
}
