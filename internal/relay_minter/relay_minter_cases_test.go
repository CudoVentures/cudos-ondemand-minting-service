package relayminter

import (
	"errors"
	"testing"

	"github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func buildTestCases(t *testing.T, encodingConfig *params.EncodingConfig, wallet sdk.AccAddress) []testCase {
	buyer1, err := sdk.AccAddressFromBech32("cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg")
	require.NoError(t, err)

	return []testCase{
		{
			name:                "ShouldReturnNoErrorWhenNilTxsResult",
			inputTxs:            nil,
			expectedError:       nil,
			expectedLogOutput:   "",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipWhenBankSendMemoIsEmpty",

			inputTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"",
			}, encodingConfig),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: memo not set in transaction ()",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipWhenBankSendMemoIsNotJSON",

			inputTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"nftuid#1",
			}, encodingConfig),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: unmarshaling memo (nftuid#1) failed: invalid character 'f' in literal null (expecting 'u')",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipIfTransactionContainsMoreThanOneMessage",

			inputTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: received bank send tx should contain exactly one message but instead it contains 2",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipIfMessageIsNotMsgSend",

			inputTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgMultiSend(nil, nil),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: not valid bank send",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipWhenBankSendIsNotSendToTheManagedWallet",

			inputTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: bank send receiver (cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg) is not the wallet (cudos1a326k254fukx9jlp0h3fwcr2ymjgludzum67dv)",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipWhenBankSendHasMultipleCoinsSent",

			inputTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)), sdk.NewCoin("ucudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: bank send should have single coin sent instead got 100acudos,100ucudos",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipWhenBankSendIsWithNonPaymentDenom",

			inputTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("ucudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: bank send invalid payment denom, expected acudos but got ucudos",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShoulSkipIfEmptyDataIsReturnedForGivenNFT",

			inputTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#0\"}",
			}, encodingConfig),

			expectedError:       errors.New("failed to mint: nft (nftuid#0) has invalid status ()"),
			expectedLogOutput:   "during check if minted, message was not mint msg\r\nrefund bank send from expected cudos1a326k254fukx9jlp0h3fwcr2ymjgludzum67dv but actual is cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg\r\nduring refund received amount without gas (-901) is smaller than minimum refund amount (5000000000000000000)",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
	}
}

type testCase struct {
	name                string
	inputTxs            *ctypes.ResultTxSearch
	expectedError       error
	expectedLogOutput   string
	expectedOutputMemos []string
	expectedOutputMsgs  []sdk.Msg
}

func buildTestResultTxSearch(t *testing.T, msgs [][]sdk.Msg, memos []string, encodingConfig *params.EncodingConfig) *ctypes.ResultTxSearch {
	require.Len(t, msgs, len(memos))

	resultTxSearch := ctypes.ResultTxSearch{}

	for i := range msgs {
		resultTx := &ctypes.ResultTx{
			Height: int64(i),
			Tx:     buildTestTx(t, msgs[i], memos[i], encodingConfig),
		}

		resultTxSearch.Txs = append(resultTxSearch.Txs, resultTx)
	}

	return &resultTxSearch
}

func buildTestTx(t *testing.T, msgs []sdk.Msg, memo string, encodingConfig *params.EncodingConfig) []byte {

	txBuilder := encodingConfig.TxConfig.NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(msgs...))
	txBuilder.SetMemo(memo)

	txBytes, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	return txBytes
}
