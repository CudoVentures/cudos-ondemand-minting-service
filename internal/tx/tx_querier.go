package tx

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
)

func NewTxQuerier(node txSearcher) *txQuerier {
	return &txQuerier{node: node}
}

func (tq *txQuerier) query(ctx context.Context, query string) (*ctypes.ResultTxSearch, error) {
	var allResults *ctypes.ResultTxSearch = nil

	for page := 1; ; page += 1 {
		results, err := tq.node.TxSearch(ctx, query, true, &page, &itemsPerPage, "asc")
		if err != nil {
			return nil, fmt.Errorf("tx search (%s) failed: %s", query, err)
		}

		if allResults == nil {
			allResults = results
		} else {
			allResults.Txs = append(allResults.Txs, results.Txs...)
			allResults.TotalCount = results.TotalCount
		}

		from := (page - 1) * itemsPerPage
		to := page*itemsPerPage - 1

		// there are no events so we can break it directly
		if allResults.TotalCount == 0 {
			break
		}
		// the "+1" in from and to are because from,to are 0th based, while TotalCount is 1 based
		if from+1 <= allResults.TotalCount && allResults.TotalCount <= to+1 {
			break
		}
	}

	return allResults, nil
}

func (tq *txQuerier) QueryLegacy(ctx context.Context, query []*TxQuerierLegacyParams, heights ...int64) (*ctypes.ResultTxSearch, error) {
	resultTxMapResult := make(map[string]*ctypes.ResultTx)

	heightFrom := int64(-1)
	heightTo := int64(-1)
	if len(heights) >= 1 {
		heightFrom = heights[0]
	}
	if len(heights) >= 2 {
		heightFrom = heights[1]
	}

	for i, param := range query {
		queryBuilder := []string{param.Key + "='" + param.Value + "'"}
		if heightFrom != -1 {
			queryBuilder = append(queryBuilder, "tx.height>="+strconv.FormatInt(heightFrom, 10))
		}
		if heightTo != -1 {
			queryBuilder = append(queryBuilder, "tx.height<="+strconv.FormatInt(heightTo, 10))
		}

		resultTxSearch, err := tq.query(ctx, strings.Join(queryBuilder, " AND "))
		if err != nil {
			return nil, err
		}

		if i == 0 {
			for _, resultTx := range resultTxSearch.Txs {
				resultTxMapResult[resultTx.Hash.String()] = resultTx
			}
		} else {
			resultTxMap := make(map[string]*ctypes.ResultTx)
			for _, resultTx := range resultTxSearch.Txs {
				hash := resultTx.Hash.String()
				_, found := resultTxMapResult[hash]
				if found {
					resultTxMap[hash] = resultTx
				}
			}
			resultTxMapResult = resultTxMap
		}

		// no point to conitinue if resulting map already does not have any elements in
		if len(resultTxMapResult) == 0 {
			break
		}
	}

	resultTx := make([]*ctypes.ResultTx, 0, len(resultTxMapResult))
	for _, value := range resultTxMapResult {
		resultTx = append(resultTx, value)
	}

	result := &ctypes.ResultTxSearch{
		Txs:        resultTx,
		TotalCount: len(resultTx),
	}

	return result, nil
}

var itemsPerPage = 100

type txQuerier struct {
	node txSearcher
}

type txSearcher interface {
	TxSearch(ctx context.Context, query string, prove bool, page, perPage *int, orderBy string) (*ctypes.ResultTxSearch, error)
}

type TxQuerierLegacyParams struct {
	Key   string
	Value string
}
