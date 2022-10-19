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
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
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
	},
		map[string]error{
			"nftuid#4": errors.New("not found"),
		},
		map[string]error{
			"nftuid#3": errors.New("not found"),
		})
	mockLogger := newMockLogger()

	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, mockStatesStorage,
		mockTokenisedInfraClient, privKey, grpc.GRPCConnector{}, rpc.RPCConnector{}, tx.NewTxCoder(&encodingConfig))

	testCases := buildTestCases(t, &encodingConfig, relayMinter.walletAddress)

	for i := 0; i < len(testCases); i++ {

		relayMinter.txQuerier = newMockTxQuerier(testCases[i].receivedBankSendTxs, testCases[i].mintTxs,
			testCases[i].sentBankSendTxs, testCases[i].failMintTxsQuery)
		mts := newMockTxSender(testCases[i].failAllSendTx)
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
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, cfg, nil, nil, privKey, &grpcConnector, rpc.RPCConnector{}, tx.NewTxCoder(&encodingConfig))
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
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, cfg, nil, nil, privKey, grpc.GRPCConnector{}, &rpcConnector, tx.NewTxCoder(&encodingConfig))
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
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil, tx.NewTxCoder(&encodingConfig))

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
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil, tx.NewTxCoder(&encodingConfig))

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
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil, tx.NewTxCoder(&encodingConfig))

	sendTxFail := errors.New("failed to send tx")
	mcts := mockCallsTxSender{}
	mcts.On("EstimateGas", mock.Anything, mock.Anything, mock.Anything).Return(model.GasResult{GasLimit: 1}, nil)
	mcts.On("SendTx", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", sendTxFail)
	relayMinter.txSender = &mcts

	nftData := model.NFTData{
		Status: model.ApprovedNFTStatus,
		Price:  sdk.NewCoin("acudos", sdk.NewIntFromUint64(10)),
	}

	err = relayMinter.mint(context.Background(), "uid", "cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg",
		nftData, sdk.NewCoin("acudos", sdk.NewIntFromUint64(10000000000000000)))
	require.Equal(t, sendTxFail, err)
}

func TestShouldFailRefundIfParsingWalletAddressFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil, nil)
	relayMinter.txQuerier = newMockTxQuerier(nil, nil, nil, false)
	relayMinter.walletAddress = sdk.AccAddress{}

	err = relayMinter.refund(context.Background(), "txHash", "refundReceiver", sdk.NewCoin("acudos", sdk.NewInt(0)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid wallet address")
}

func TestShouldFailRefundIfParsingRefundReceiverAddressFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil, nil)
	relayMinter.txQuerier = newMockTxQuerier(nil, nil, nil, false)

	err = relayMinter.refund(context.Background(), "txHash", "refundReceiver", sdk.NewCoin("acudos", sdk.NewInt(0)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid refund receiver address")
}

func TestShouldFailRefundIfEstimateGasFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil, nil)
	relayMinter.txQuerier = newMockTxQuerier(nil, nil, nil, false)

	gasEstimateFail := errors.New("failed to estimate gas")
	mcts := mockCallsTxSender{}
	mcts.On("EstimateGas", mock.Anything, mock.Anything, mock.Anything).Return(model.GasResult{}, gasEstimateFail)
	relayMinter.txSender = &mcts

	err = relayMinter.refund(context.Background(), "txHash", "cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg", sdk.NewCoin("acudos", sdk.NewInt(0)))
	require.Equal(t, gasEstimateFail, err)
}

func TestShouldFailRefundIfIsRefundFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil, nil)
	txQuerier := mockCallsTxQuerier{}
	failedQuery := errors.New("failed query")
	txQuerier.On("Query", mock.Anything, mock.Anything).Return(&ctypes.ResultTxSearch{}, failedQuery)
	relayMinter.txQuerier = &txQuerier

	err = relayMinter.refund(context.Background(), "txHash", "cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg", sdk.NewCoin("acudos", sdk.NewInt(0)))
	require.Equal(t, failedQuery, err)
}

func TestShouldFailIsMintedIfQueryFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil, nil)

	txQuerier := mockCallsTxQuerier{}
	failedQuery := errors.New("failed query")
	txQuerier.On("Query", mock.Anything, mock.Anything).Return(&ctypes.ResultTxSearch{}, failedQuery)
	relayMinter.txQuerier = &txQuerier

	isMinted, err := relayMinter.isMinted(context.Background(), "testuid")
	require.Equal(t, false, isMinted)
	require.Equal(t, failedQuery, err)
}

func TestIsMintedShouldLogErrorIfDecodingTxFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, nil, nil, privKey, nil, nil, nil)

	txQuerier := mockCallsTxQuerier{}
	txQuerier.On("Query", mock.Anything, mock.Anything).Return(&ctypes.ResultTxSearch{
		Txs: []*ctypes.ResultTx{
			{},
		},
	}, nil)
	relayMinter.txQuerier = &txQuerier

	mctc := mockTxCoder{}
	mctc.On("Decode", mock.Anything, mock.Anything).Return(&txWithoutMemo{}, nil)
	relayMinter.txCoder = &mctc

	isMinted, err := relayMinter.isMinted(context.Background(), "testuid")
	require.Equal(t, false, isMinted)
	require.NoError(t, err)
	require.Equal(t, "during check if minted, decoding tx () failed: invalid transaction () type", mockLogger.output)
}

func TestShouldRetryRelayingIfRelayFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockLogger := newMockLogger()
	encodingConfig := encodingconfig.MakeEncodingConfig()
	cfg := config.Config{
		PaymentDenom:  "acudos",
		RelayInterval: 1 * time.Second,
		RetryInterval: 1 * time.Second,
		MaxRetries:    10,
		ChainID:       "cudos-local-network",
		ChainRPC:      "http://127.0.0.1:26657",
		ChainGRPC:     "127.0.0.1:9090",
	}

	failedGettingState := errors.New("failed getting state")

	mcss := mockCallsStateStorage{}
	mcss.On("GetState").Return(model.State{}, failedGettingState)

	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, cfg, &mcss, nil, privKey, grpc.GRPCConnector{}, rpc.RPCConnector{}, tx.NewTxCoder(&encodingConfig))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go relayMinter.Start(ctx)
	<-ctx.Done()

	require.Contains(t, mockLogger.output, failedGettingState.Error())
}

func TestShouldFailRelayIfQueryFails(t *testing.T) {
	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockStatesStorage := newMockState()

	relayMinter := NewRelayMinter(nil, nil, config.Config{}, mockStatesStorage, nil, privKey, nil, nil, nil)
	txQuerier := mockCallsTxQuerier{}
	failedQuery := errors.New("failed query")
	txQuerier.On("Query", mock.Anything, mock.Anything).Return(&ctypes.ResultTxSearch{}, failedQuery)
	relayMinter.txQuerier = &txQuerier

	err = relayMinter.relay(context.Background())
	require.Equal(t, failedQuery, err)
}

type mockCallsStateStorage struct {
	mock.Mock
}

func (mcss *mockCallsStateStorage) GetState() (model.State, error) {
	args := mcss.Called()
	return args.Get(0).(model.State), args.Error(1)
}

func (mcss *mockCallsStateStorage) UpdateState(state model.State) error {
	args := mcss.Called(state)
	return args.Error(0)
}

type mockTxCoder struct {
	mock.Mock
}

func (mctc *mockTxCoder) Decode(tx tmtypes.Tx) (sdk.Tx, error) {
	args := mctc.Called(tx)
	return args.Get(0).(sdk.Tx), args.Error(1)
}

type txWithoutMemo struct {
}

func (twm *txWithoutMemo) GetMsgs() []sdk.Msg {
	return nil
}

func (twm *txWithoutMemo) ValidateBasic() error {
	return nil
}

type mockCallsTxQuerier struct {
	mock.Mock
}

func (mctq *mockCallsTxQuerier) Query(ctx context.Context, query string) (*ctypes.ResultTxSearch, error) {
	args := mctq.Called(ctx, query)
	return args.Get(0).(*ctypes.ResultTxSearch), args.Error(1)
}

type mockCallsTxSender struct {
	mock.Mock
}

func (mcts *mockCallsTxSender) EstimateGas(ctx context.Context, msgs []sdk.Msg, memo string) (model.GasResult, error) {
	args := mcts.Called(ctx, msgs, memo)
	return args.Get(0).(model.GasResult), args.Error(1)
}

func (mcts *mockCallsTxSender) SendTx(ctx context.Context, msgs []sdk.Msg, memo string, gasResult model.GasResult) (string, error) {
	args := mcts.Called(ctx, msgs, memo, gasResult)
	return args.String(0), args.Error(1)
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
