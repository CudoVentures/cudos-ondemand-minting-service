package relayminter

import (
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

func newTokenisedInfraClient() *mockTokenisedInfraClient {
	return &mockTokenisedInfraClient{}
}

func (mtic *mockTokenisedInfraClient) GetNFTData(uid string) (model.NFTData, error) {
	return model.NFTData{}, nil
}

func (mtic *mockTokenisedInfraClient) MarkMintedNFT(uid string) error {
	return nil
}

type mockTokenisedInfraClient struct {
}

func newMockTxQuerier(queryResult *ctypes.ResultTxSearch) *mockTxQuerier {
	return &mockTxQuerier{queryResult: queryResult}
}

func (mq *mockTxQuerier) Query(query string) (*ctypes.ResultTxSearch, error) {
	return mq.queryResult, nil
}

type mockTxQuerier struct {
	queryResult *ctypes.ResultTxSearch
}

func newMockTxSender() *mockTxSender {
	return &mockTxSender{
		outputMemos: []string{},
		outputMsgs:  []sdk.Msg{},
	}
}

func (mts *mockTxSender) EstimateGas(msgs []sdk.Msg) (model.GasResult, error) {
	return model.GasResult{
		FeeAmount: mockFeeAmount,
		GasLimit:  mockGasLimit,
	}, nil
}

func (mts *mockTxSender) SendTx(msgs []sdk.Msg, memo string) error {
	mts.outputMsgs = append(mts.outputMsgs, msgs...)
	mts.outputMemos = append(mts.outputMemos, memo)
	return nil
}

type mockTxSender struct {
	outputMemos []string
	outputMsgs  []sdk.Msg
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
