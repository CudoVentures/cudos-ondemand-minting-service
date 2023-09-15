package tx

import (
	"context"
	"errors"
	"fmt"
	"testing"

	encodingconfig "github.com/CudoVentures/cudos-ondemand-minting-service/internal/encoding_config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/key"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	client "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	auth "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestShouldFailSendTxIfBroadcastFails(t *testing.T) {
	broadcastFailed := errors.New("broadcast failed")
	mTxClient := mockTxClient{}
	mTxClient.On("BroadcastTx", mock.Anything, mock.Anything, mock.Anything).Return(&txtypes.BroadcastTxResponse{}, broadcastFailed)

	mAccInfoClient := mockAccountInfoClient{}
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, nil)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	txSender := NewTxSender(&mTxClient, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, NewTxSigner(&encodingConfig, privKey))

	_, err = txSender.SendTx(context.Background(), []types.Msg{}, "", model.GasResult{})
	require.Equal(t, broadcastFailed, err)
}

func TestShouldFailSendTxIfBrodcastReturnsNilTxResponse(t *testing.T) {
	mTxClient := mockTxClient{}
	mTxClient.On("BroadcastTx", mock.Anything, mock.Anything, mock.Anything).Return(&txtypes.BroadcastTxResponse{TxResponse: nil}, nil)

	mAccInfoClient := mockAccountInfoClient{}
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, nil)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	txSender := NewTxSender(&mTxClient, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, NewTxSigner(&encodingConfig, privKey))

	_, err = txSender.SendTx(context.Background(), []types.Msg{}, "", model.GasResult{})
	require.Equal(t, errors.New("broadcasting of tx failed: "), err)
}

func TestShouldFailSendTxIfBrodcastReturnsTxResponseWithNonZeroCode(t *testing.T) {
	mTxClient := mockTxClient{}
	response := txtypes.BroadcastTxResponse{
		TxResponse: &types.TxResponse{Code: 1},
	}
	mTxClient.On("BroadcastTx", mock.Anything, mock.Anything, mock.Anything).Return(&response, nil)

	mAccInfoClient := mockAccountInfoClient{}
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, nil)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	txSender := NewTxSender(&mTxClient, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, NewTxSigner(&encodingConfig, privKey))

	_, err = txSender.SendTx(context.Background(), []types.Msg{}, "", model.GasResult{})
	require.Equal(t, fmt.Errorf("broadcasting of tx failed: %+v", &response), err)
}

func TestShouldFailSendTxIfBuildingTxFails(t *testing.T) {
	mAccInfoClient := mockAccountInfoClient{}
	failedQueryInfo := errors.New("failed to query info")
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, failedQueryInfo)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	txSender := NewTxSender(&mockTxClient{}, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, NewTxSigner(&encodingConfig, privKey))

	_, err = txSender.SendTx(context.Background(), []types.Msg{}, "", model.GasResult{})
	require.Equal(t, failedQueryInfo, err)
}

func TestShouldFailEstimateGasIfSimulateFails(t *testing.T) {
	mTxClient := mockTxClient{}
	failedSimulate := errors.New("simulate failed")
	mTxClient.On("Simulate", mock.Anything, mock.Anything, mock.Anything).Return(&tx.SimulateResponse{}, failedSimulate)

	mAccInfoClient := mockAccountInfoClient{}
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, nil)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	txSender := NewTxSender(&mTxClient, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, NewTxSigner(&encodingConfig, privKey))

	_, err = txSender.EstimateGas(context.Background(), []types.Msg{}, "")
	require.Equal(t, failedSimulate, err)
}

func TestShouldFailEstimateGasIfSimulateReturnsNilGasInfo(t *testing.T) {
	mTxClient := mockTxClient{}
	mTxClient.On("Simulate", mock.Anything, mock.Anything, mock.Anything).Return(&tx.SimulateResponse{GasInfo: nil}, nil)

	mAccInfoClient := mockAccountInfoClient{}
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, nil)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	txSender := NewTxSender(&mTxClient, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, NewTxSigner(&encodingConfig, privKey))

	_, err = txSender.EstimateGas(context.Background(), []types.Msg{}, "")
	require.Equal(t, errors.New("simulation result with no gas info"), err)
}

func TestShouldFailEstimateGasIfBuildingTxFail(t *testing.T) {
	mAccInfoClient := mockAccountInfoClient{}
	failedQueryInfo := errors.New("failed to query info")
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, failedQueryInfo)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	txSender := NewTxSender(&mockTxClient{}, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, NewTxSigner(&encodingConfig, privKey))

	_, err = txSender.EstimateGas(context.Background(), []types.Msg{}, "")
	require.Equal(t, failedQueryInfo, err)
}

func TestShouldFailBuildTxIfGenTxFails(t *testing.T) {
	mAccInfoClient := mockAccountInfoClient{}
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, nil)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	failedSetMsgs := errors.New("failed to set msgs")
	mTxSigner := mockTxSigner{}
	mTxSigner.On("SetMsgs", mock.Anything, mock.Anything).Return(failedSetMsgs)

	txSender := NewTxSender(&mockTxClient{}, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, &mTxSigner)

	_, err = txSender.buildTx(context.Background(), []types.Msg{}, "", model.GasResult{})
	require.Equal(t, failedSetMsgs, err)
}

func TestShouldGenTxShouldFailIfSetSignaturesFails(t *testing.T) {
	mAccInfoClient := mockAccountInfoClient{}
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, nil)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	failedSetSignatures := errors.New("failed to set signatures")
	mTxSigner := mockTxSigner{}
	mTxSigner.On("SetMsgs", mock.Anything, mock.Anything).Return(nil)
	mTxSigner.On("SetSignatures", mock.Anything, mock.Anything).Return(failedSetSignatures)

	txSender := NewTxSender(&mockTxClient{}, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, &mTxSigner)

	addr, err := sdk.AccAddressFromBech32("cosmos1a326k254fukx9jlp0h3fwcr2ymjgludza2npne")
	require.NoError(t, err)

	bankSendMsg := banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000))))

	_, err = txSender.genTx([]types.Msg{bankSendMsg}, "", sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewInt(1))), 1, 1, 1)
	require.Equal(t, failedSetSignatures, err)
}

func TestShouldGenTxShouldFailIfGetSignBytesFails(t *testing.T) {
	mAccInfoClient := mockAccountInfoClient{}
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, nil)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	failedGetSignBytes := errors.New("failed to get sign bytes")
	mTxSigner := mockTxSigner{}
	mTxSigner.On("SetMsgs", mock.Anything, mock.Anything).Return(nil)
	mTxSigner.On("SetSignatures", mock.Anything, mock.Anything).Return(nil)
	mTxSigner.On("GetSignBytes", mock.Anything, mock.Anything, mock.Anything).Return([]byte{}, failedGetSignBytes)

	txSender := NewTxSender(&mockTxClient{}, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, &mTxSigner)

	addr, err := sdk.AccAddressFromBech32("cosmos1a326k254fukx9jlp0h3fwcr2ymjgludza2npne")
	require.NoError(t, err)

	bankSendMsg := banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000))))

	_, err = txSender.genTx([]types.Msg{bankSendMsg}, "", sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewInt(1))), 1, 1, 1)
	require.Equal(t, failedGetSignBytes, err)
}

func TestShouldGenTxShouldFailIfSignFails(t *testing.T) {
	mAccInfoClient := mockAccountInfoClient{}
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, nil)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	failedSign := errors.New("failed to sign")
	mTxSigner := mockTxSigner{}
	mTxSigner.On("SetMsgs", mock.Anything, mock.Anything).Return(nil)
	mTxSigner.On("SetSignatures", mock.Anything, mock.Anything).Return(nil)
	mTxSigner.On("GetSignBytes", mock.Anything, mock.Anything, mock.Anything).Return([]byte{}, nil)
	mTxSigner.On("Sign", mock.Anything).Return([]byte{}, failedSign)

	txSender := NewTxSender(&mockTxClient{}, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, &mTxSigner)

	addr, err := sdk.AccAddressFromBech32("cosmos1a326k254fukx9jlp0h3fwcr2ymjgludza2npne")
	require.NoError(t, err)

	bankSendMsg := banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000))))

	_, err = txSender.genTx([]types.Msg{bankSendMsg}, "", sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewInt(1))), 1, 1, 1)
	require.Equal(t, failedSign, err)
}

func TestShouldGenTxShouldFailIfSecondSetSignaturesFails(t *testing.T) {
	mAccInfoClient := mockAccountInfoClient{}
	mAccInfoClient.On("QueryInfo", mock.Anything, mock.Anything).Return(model.AccountInfo{}, nil)

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)
	encodingConfig := encodingconfig.MakeEncodingConfig()

	failedSetSignatures := errors.New("failed to set signatues")
	mTxSigner := mockTxSigner{}
	mTxSigner.On("SetMsgs", mock.Anything, mock.Anything).Return(nil)
	mTxSigner.On("SetSignatures", mock.Anything, mock.Anything).Return(nil).Once()
	mTxSigner.On("SetSignatures", mock.Anything, mock.Anything).Return(failedSetSignatures).Once()
	mTxSigner.On("GetSignBytes", mock.Anything, mock.Anything, mock.Anything).Return([]byte{}, nil)
	mTxSigner.On("Sign", mock.Anything).Return([]byte{}, nil)

	txSender := NewTxSender(&mockTxClient{}, &mAccInfoClient, &encodingConfig, privKey, "cudos-local-network", "acudos", 1, 1.3, &mTxSigner)

	addr, err := sdk.AccAddressFromBech32("cosmos1a326k254fukx9jlp0h3fwcr2ymjgludza2npne")
	require.NoError(t, err)

	bankSendMsg := banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000))))

	_, err = txSender.genTx([]types.Msg{bankSendMsg}, "", sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewInt(1))), 1, 1, 1)
	require.Equal(t, failedSetSignatures, err)
}

type mockTxSigner struct {
	mock.Mock
}

func (mts *mockTxSigner) SetMsgs(tx client.TxBuilder, msgs ...sdk.Msg) error {
	args := mts.Called(tx, msgs)
	return args.Error(0)
}

func (mts *mockTxSigner) SetSignatures(tx client.TxBuilder, signatures ...signingtypes.SignatureV2) error {
	args := mts.Called(tx, signatures)
	return args.Error(0)
}

func (mts *mockTxSigner) GetSignBytes(mode signing.SignMode, data auth.SignerData, tx sdk.Tx) ([]byte, error) {
	args := mts.Called(mode, data, tx)
	return args.Get(0).([]byte), args.Error(1)
}

func (mts *mockTxSigner) Sign(msg []byte) ([]byte, error) {
	args := mts.Called(msg)
	return args.Get(0).([]byte), args.Error(1)
}

type mockTxClient struct {
	mock.Mock
}

func (mtc *mockTxClient) Simulate(ctx context.Context, in *txtypes.SimulateRequest, opts ...grpc.CallOption) (*txtypes.SimulateResponse, error) {
	args := mtc.Called(ctx, in, opts)
	return args.Get(0).(*txtypes.SimulateResponse), args.Error(1)
}

func (mtc *mockTxClient) BroadcastTx(ctx context.Context, in *txtypes.BroadcastTxRequest, opts ...grpc.CallOption) (*txtypes.BroadcastTxResponse, error) {
	args := mtc.Called(ctx, in, opts)
	return args.Get(0).(*txtypes.BroadcastTxResponse), args.Error(1)
}

func (mtc *mockTxClient) GetTx(ctx context.Context, in *txtypes.GetTxRequest, opts ...grpc.CallOption) (*txtypes.GetTxResponse, error) {
	return &tx.GetTxResponse{
		Tx: nil,
		TxResponse: &sdk.TxResponse{
			Code: 0,
		},
	}, nil
}

type mockAccountInfoClient struct {
	mock.Mock
}

func (maic *mockAccountInfoClient) QueryInfo(ctx context.Context, address string) (model.AccountInfo, error) {
	args := maic.Called(ctx, address)
	return args.Get(0).(model.AccountInfo), args.Error(1)
}

const walletMnemonic = "rebel wet poet torch carpet gaze axis ribbon approve depend inflict menu"
