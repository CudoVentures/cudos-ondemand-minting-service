package relayminter

import (
	"context"
	"testing"

	cudosapp "github.com/CudoVentures/cudos-node/app"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	encodingconfig "github.com/CudoVentures/cudos-ondemand-minting-service/internal/encoding_config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/key"
	"github.com/stretchr/testify/require"
)

func TestRelayMinter(t *testing.T) {
	cudosapp.SetConfig()
	encodingConfig := encodingconfig.MakeEncodingConfig()

	privKey, err := key.PrivKeyFromMnemonic(walletMnemonic)
	require.NoError(t, err)

	mockStatesStorage := newMockState()
	mockTokenisedInfraClient := newTokenisedInfraClient()
	mockLogger := newMockLogger()

	relayMinter := NewRelayMinter(mockLogger, &encodingConfig, config.Config{PaymentDenom: "acudos"}, mockStatesStorage, mockTokenisedInfraClient, privKey)

	testCases := buildTestCases(t, &encodingConfig, relayMinter.walletAddress)

	for i := 0; i < len(testCases); i++ {

		relayMinter.txQuerier = newMockTxQuerier(testCases[i].inputTxs)
		mts := newMockTxSender()
		relayMinter.txSender = mts

		mockLogger = newMockLogger()

		relayMinter.logger = mockLogger

		err := relayMinter.relay(context.Background())

		require.Equal(t, testCases[i].expectedError, err, testCases[i].name)
		require.Equal(t, testCases[i].expectedLogOutput, mockLogger.output, testCases[i].name)

		verifyOutputMemos(t, testCases[i], mts)
		verifyOutputMsgs(t, testCases[i], mts)
	}
}

func verifyOutputMemos(t *testing.T, tc testCase, mts *mockTxSender) {
	require.Len(t, tc.expectedOutputMemos, len(mts.outputMemos), tc.name)
	require.Equal(t, tc.expectedOutputMemos, mts.outputMemos, tc.name)
}

func verifyOutputMsgs(t *testing.T, tc testCase, mts *mockTxSender) {
	require.Len(t, tc.expectedOutputMsgs, len(mts.outputMsgs), tc.name)
	require.Equal(t, tc.expectedOutputMsgs, mts.outputMsgs, tc.name)
}

const walletMnemonic = "rebel wet poet torch carpet gaze axis ribbon approve depend inflict menu"
