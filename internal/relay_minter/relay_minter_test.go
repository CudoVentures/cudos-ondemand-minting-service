package relayminter

import (
	"context"
	"errors"
	"testing"
	"time"

	cudosapp "github.com/CudoVentures/cudos-node/app"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	encodingconfig "github.com/CudoVentures/cudos-ondemand-minting-service/internal/encoding_config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/grpc"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/key"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/rpc"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ggrpc "google.golang.org/grpc"
)

func TestRelay(t *testing.T) {
	cudosapp.SetConfig()
	encodingConfig := encodingconfig.MakeEncodingConfig()

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockStatesStorage := newMockState()
	mockTokenisedInfraClient := newTokenisedInfraClient(map[string]model.NFTData{
		"nftuid#1": {
			Price:   sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)),
			Name:    "test nft name",
			Uri:     "test nft uri",
			Data:    "test nft data",
			DenomID: "testdenom",
			Status:  model.ApprovedNFTStatus,
		},
		"nftuid#2": {
			Price:   sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)),
			Name:    "test nft name",
			Uri:     "test nft uri",
			Data:    "test nft data",
			DenomID: "testdenom",
			Status:  model.RejectedNFTStatus,
		},
		"nftuid#3": {
			Price:   sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)),
			Name:    "test nft name",
			Uri:     "test nft uri",
			Data:    "test nft data",
			DenomID: "testdenom",
			Status:  model.ApprovedNFTStatus,
		},
	}, map[string]error{
		"nftuid#3": errors.New("not found"),
	})
	mockLogger := newMockLogger()

	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, mockStatesStorage,
		mockTokenisedInfraClient, privKey, grpc.GRPCConnector{}, rpc.RPCConnector{})

	testCases := buildTestCases(t, &encodingConfig, relayMinter.walletAddress)

	for i := 0; i < len(testCases); i++ {

		relayMinter.txQuerier = newMockTxQuerier(testCases[i].receivedBankSendTxs, testCases[i].mintTxs, testCases[i].sentBankSendTxs)
		mts := newMockTxSender()
		relayMinter.txSender = mts

		mockLogger = newMockLogger()

		relayMinter.logger = mockLogger

		err := relayMinter.relay(context.Background())

		require.Equal(t, testCases[i].expectedError, err, testCases[i].name)
		require.Equal(t, testCases[i].expectedLogOutput, mockLogger.output, testCases[i].name)

		require.Equal(t, testCases[i].expectedOutputMemos, mts.outputMemos, testCases[i].name)
		require.Equal(t, testCases[i].expectedOutputMsgs, mts.outputMsgs, testCases[i].name)
	}
}

func TestShouldRetryIfGRPCConnectionFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	cfg := config.Config{
		PaymentDenom:  "acudos",
		RetryInterval: 1 * time.Second,
		MaxRetries:    10,
	}
	grpcConnector := mockGRPCConnector{}
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, cfg, nil, nil, privKey, &grpcConnector, rpc.RPCConnector{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go relayMinter.Start(ctx)
	<-ctx.Done()
	require.Greater(t, grpcConnector.connectsCount, 2)
}

func TestShouldRetryIfRPCConnectionFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	cfg := config.Config{
		PaymentDenom:  "acudos",
		RetryInterval: 1 * time.Second,
		MaxRetries:    10,
	}
	rpcConnector := mockRPCConnector{}
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, cfg, nil, nil, privKey, grpc.GRPCConnector{}, &rpcConnector)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go relayMinter.Start(ctx)
	<-ctx.Done()
	require.Greater(t, rpcConnector.connectsCount, 2)
}

func TestShouldFailMintIfEstimateGasFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil)

	gasEstimateFail := errors.New("failed to estimate gas")
	mcts := mockCallsTxSender{}
	mcts.On("EstimateGas", mock.Anything, mock.Anything, mock.Anything).Return(model.GasResult{}, gasEstimateFail)
	relayMinter.txSender = &mcts

	err = relayMinter.mint(context.Background(), "uid", "cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg",
		model.NFTData{Status: model.ApprovedNFTStatus}, sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))
	require.Equal(t, gasEstimateFail, err)
}

func TestShouldFailMintIfReceivedAmountIsSmallerThanTheGasFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil)

	mcts := mockCallsTxSender{}
	mcts.On("EstimateGas", mock.Anything, mock.Anything, mock.Anything).Return(model.GasResult{GasLimit: 1}, nil)
	relayMinter.txSender = &mcts

	nftData := model.NFTData{
		Status: model.ApprovedNFTStatus,
		Price:  sdk.NewCoin("acudos", sdk.NewIntFromUint64(10)),
	}
	err = relayMinter.mint(context.Background(), "uid", "cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg",
		nftData, sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))
	require.Equal(t, errors.New("during mint received amount (100) is smaller than the gas (5000000000000)"), err)
}

func TestShouldFailMintIfSendTxFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil)

	sendTxFail := errors.New("failed to send tx")
	mcts := mockCallsTxSender{}
	mcts.On("EstimateGas", mock.Anything, mock.Anything, mock.Anything).Return(model.GasResult{GasLimit: 1}, nil)
	mcts.On("SendTx", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sendTxFail)
	relayMinter.txSender = &mcts

	nftData := model.NFTData{
		Status: model.ApprovedNFTStatus,
		Price:  sdk.NewCoin("acudos", sdk.NewIntFromUint64(10)),
	}

	err = relayMinter.mint(context.Background(), "uid", "cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg",
		nftData, sdk.NewCoin("acudos", sdk.NewIntFromUint64(10000000000000000)))
	require.Equal(t, sendTxFail, err)
}

type mockCallsTxSender struct {
	mock.Mock
}

func (mcts *mockCallsTxSender) EstimateGas(ctx context.Context, msgs []sdk.Msg, memo string) (model.GasResult, error) {
	args := mcts.Called(ctx, msgs, memo)
	return args.Get(0).(model.GasResult), args.Error(1)
}

func (mcts *mockCallsTxSender) SendTx(ctx context.Context, msgs []sdk.Msg, memo string, gasResult model.GasResult) error {
	args := mcts.Called(ctx, msgs, memo, gasResult)
	return args.Error(0)
}

func (mgc *mockGRPCConnector) MakeGRPCClient(url string) (*ggrpc.ClientConn, error) {
	mgc.connectsCount += 1
	return nil, errors.New("failed to connect")
}

type mockGRPCConnector struct {
	connectsCount int
}

func (rc *mockRPCConnector) MakeRPCClient(url string) (*rpchttp.HTTP, error) {
	rc.connectsCount += 1
	return nil, errors.New("failed to connect")
}

type mockRPCConnector struct {
	connectsCount int
}

const walletMnemonic = "rebel wet poet torch carpet gaze axis ribbon approve depend inflict menu"
