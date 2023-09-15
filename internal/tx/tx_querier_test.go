package tx

import (
	"context"
	"errors"
	"testing"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/stretchr/testify/require"
)

func TestShouldFailIfNodeTxSearchFails(t *testing.T) {
	txQuerier := NewTxQuerier(&mockTxSearcher{})
	_, err := txQuerier.Query(context.Background(), "some query")
	require.Equal(t, errors.New("tx search (some query) failed: failed tx search request"), err)
}

func (mts *mockTxSearcher) TxSearch(ctx context.Context, query string, prove bool, page, perPage *int, orderBy string) (*ctypes.ResultTxSearch, error) {
	return nil, failedTxSearchRequest
}

type mockTxSearcher struct {
}

var failedTxSearchRequest = errors.New("failed tx search request")
