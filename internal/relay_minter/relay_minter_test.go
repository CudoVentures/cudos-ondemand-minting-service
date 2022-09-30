package relayminter

import (
	"context"
	"testing"

	cudosapp "github.com/CudoVentures/cudos-node/app"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	encodingconfig "github.com/CudoVentures/cudos-ondemand-minting-service/internal/encoding_config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/key"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestRelayMinter(t *testing.T) {
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
	})
	mockLogger := newMockLogger()

	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, mockStatesStorage, mockTokenisedInfraClient, privKey)

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

const walletMnemonic = "rebel wet poet torch carpet gaze axis ribbon approve depend inflict menu"
