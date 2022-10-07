package relayminter

import (
	"context"
	"errors"
	"strings"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func newMockState() *mockState {
	return &mockState{}
}

func (ms *mockState) GetState() (model.State, error) {
	return ms.state, nil
}

func (ms *mockState) UpdateState(state model.State) error {
	ms.state = state
	return nil
}

type mockState struct {
	state model.State
}

func newTokenisedInfraClient(nftDataEntires map[string]model.NFTData, getNftDataErrors, markNftErrors map[string]error) *mockTokenisedInfraClient {
	return &mockTokenisedInfraClient{
		nftDataEntires:   nftDataEntires,
		getNftDataErrors: getNftDataErrors,
		markNftErrors:    markNftErrors,
	}
}

func (mtic *mockTokenisedInfraClient) GetNFTData(ctx context.Context, uid string) (model.NFTData, error) {
	if err, ok := mtic.getNftDataErrors[uid]; ok {
		return model.NFTData{}, err
	}

	if data, ok := mtic.nftDataEntires[uid]; ok {
		return data, nil
	}
	return model.NFTData{}, nil
}

func (mtic *mockTokenisedInfraClient) MarkMintedNFT(ctx context.Context, mintedTxHash, uid string) error {
	err, ok := mtic.markNftErrors[uid]
	if ok {
		return err
	}
	return nil
}

type mockTokenisedInfraClient struct {
	nftDataEntires   map[string]model.NFTData
	getNftDataErrors map[string]error
	markNftErrors    map[string]error
}

func newMockTxQuerier(bankSendQueryResults *ctypes.ResultTxSearch, mintQueryResults *ctypes.ResultTxSearch,
	refundQueryResults *ctypes.ResultTxSearch, failMintTxsQuery bool) *mockTxQuerier {
	return &mockTxQuerier{
		bankSendQueryResults: bankSendQueryResults,
		mintQueryResults:     mintQueryResults,
		refundQueryResults:   refundQueryResults,
		failMintTxsQuery:     failMintTxsQuery,
	}
}

func (mq *mockTxQuerier) Query(ctx context.Context, query string) (*ctypes.ResultTxSearch, error) {
	if strings.Contains(query, "tx.height") {
		return mq.bankSendQueryResults, nil
	} else if strings.Contains(query, "transfer.sender") {
		return mq.refundQueryResults, nil
	} else if strings.Contains(query, "marketplace_mint_nft") {
		if mq.failMintTxsQuery {
			return nil, errors.New("failed to query mint txs")
		}
		return mq.mintQueryResults, nil
	}

	panic("invalid query")
}

type mockTxQuerier struct {
	bankSendQueryResults *ctypes.ResultTxSearch
	mintQueryResults     *ctypes.ResultTxSearch
	refundQueryResults   *ctypes.ResultTxSearch
	failMintTxsQuery     bool
}

func newMockTxSender(failAllSendTx bool) *mockTxSender {
	return &mockTxSender{
		outputMemos:   []string{},
		outputMsgs:    []sdk.Msg{},
		failAllSendTx: failAllSendTx,
	}
}

func (mts *mockTxSender) EstimateGas(ctx context.Context, msgs []sdk.Msg, memo string) (model.GasResult, error) {
	return model.GasResult{
		FeeAmount: mockFeeAmount,
		GasLimit:  mockGasLimit,
	}, nil
}

func (mts *mockTxSender) SendTx(ctx context.Context, msgs []sdk.Msg, memo string, gasResult model.GasResult) (string, error) {
	if mts.failAllSendTx {
		return "", errors.New("failed to send tx")
	}

	mts.outputMsgs = append(mts.outputMsgs, msgs...)
	mts.outputMemos = append(mts.outputMemos, memo)
	return "", nil
}

type mockTxSender struct {
	outputMemos   []string
	outputMsgs    []sdk.Msg
	failAllSendTx bool
}

func newMockLogger() *mockLogger {
	return &mockLogger{}
}

func (ml *mockLogger) Error(err error) {
	if len(ml.output) > 0 {
		ml.output += "\r\n"
	}
	ml.output += err.Error()
}

type mockLogger struct {
	output string
}

const mockGasLimit uint64 = 1001

var mockFeeAmount = sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(mockGasLimit)))
