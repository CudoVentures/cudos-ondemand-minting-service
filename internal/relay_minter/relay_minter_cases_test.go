package relayminter

import (
	"testing"

	marketplacetypes "github.com/CudoVentures/cudos-node/x/marketplace/types"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/bytes"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func buildTestCases(t *testing.T, encodingConfig *params.EncodingConfig, wallet sdk.AccAddress) []testCase {
	buyer1, err := sdk.AccAddressFromBech32("cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg")
	require.NoError(t, err)

	return []testCase{
		{
			name:                "ShouldReturnNoErrorWhenNilTxsResult",
			receivedBankSendTxs: nil,
			expectedError:       nil,
			expectedLogOutput:   "",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipWhenBankSendMemoIsEmpty",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: memo not set in transaction ()",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipWhenBankSendMemoIsNotJSON",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"nftuid#1",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: unmarshaling memo (nftuid#1) failed: invalid character 'f' in literal null (expecting 'u')",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipIfTransactionContainsMoreThanOneMessage",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: received bank send tx should contain exactly one message but instead it contains 2",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipIfMessageIsNotMsgSend",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgMultiSend(nil, nil),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: not valid bank send",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipWhenBankSendIsNotSendToTheManagedWallet",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: bank send receiver (cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg) is not the wallet (cudos1a326k254fukx9jlp0h3fwcr2ymjgludzum67dv)",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipWhenBankSendHasMultipleCoinsSent",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)), sdk.NewCoin("ucudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: bank send should have single coin sent instead got 100acudos,100ucudos",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldSkipWhenBankSendIsWithNonPaymentDenom",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("ucudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: bank send invalid payment denom, expected acudos but got ucudos",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShoulSkipRefundIfWhenSubtractingGasFeesTheAmountIsNegative",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(100)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#0\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "during refund received amount without gas (-5004999999999900) is smaller than minimum refund amount (5000000000000000000)\r\nfailed to mint: nft (nftuid#0) was not found",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldHaveSingleMessageInMintTransaction",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),
			mintTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					marketplacetypes.NewMsgMintNft(wallet.String(), "", buyer1.String(), "", "", "", "nftuid#1", sdk.NewCoin("acudos", sdk.NewIntFromUint64(100))),
					marketplacetypes.NewMsgMintNft(wallet.String(), "", buyer1.String(), "", "", "", "nftuid#1", sdk.NewCoin("acudos", sdk.NewIntFromUint64(100))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "during check if minted, expected one message but got 2\r\nfailed to mint: during mint received amount without gas (7994995000000000000) is smaller than price (8000000000000000000)",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Sub(sdk.NewIntFromUint64(5005000000000000))))),
			},
		},
		{
			name: "ShouldHaveSingleMessageInMintTransactionWhichShouldBeMintMsg",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),
			mintTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					marketplacetypes.NewMsgBuyNft(wallet.String(), 1),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "during check if minted, message was not mint msg\r\nfailed to mint: during mint received amount without gas (7994995000000000000) is smaller than price (8000000000000000000)",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Sub(sdk.NewIntFromUint64(5005000000000000))))),
			},
		},
		{
			name: "ShouldHaveWalletAsCreatorOfMintMsg",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),
			mintTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					marketplacetypes.NewMsgMintNft(buyer1.String(), "", buyer1.String(), "", "", "", "nftuid#1", sdk.NewCoin("acudos", sdk.NewIntFromUint64(100))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "during check if minted, creator (cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg) of the mint msg is not equal to wallet (cudos1a326k254fukx9jlp0h3fwcr2ymjgludzum67dv)\r\nfailed to mint: during mint received amount without gas (7994995000000000000) is smaller than price (8000000000000000000)",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Sub(sdk.NewIntFromUint64(5005000000000000))))),
			},
		},
		{
			name: "ShouldSkipMintingIfAlreadyMinted",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),
			mintTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					marketplacetypes.NewMsgMintNft(wallet.String(), "", buyer1.String(), "", "", "", "nftuid#1", sdk.NewCoin("acudos", sdk.NewIntFromUint64(100))),
				},
			}, []string{
				"nftuid#1",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "already minted nftuid#1",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Sub(sdk.NewIntFromUint64(5005000000000000))))),
			},
		},
		{
			name: "ShouldSkipIfReceivedInvalidTx",

			receivedBankSendTxs: &ctypes.ResultTxSearch{Txs: []*ctypes.ResultTx{
				{Tx: []byte("invalid tx")},
			}},

			expectedError:       nil,
			expectedLogOutput:   "getting received bank send info failed: getting received bank info: decoding transaction () result failed: expected 2 wire type, got 1: tx parse error",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldHaveValidTxDuringIsRefundCheck",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),
			sentBankSendTxs: &ctypes.ResultTxSearch{Txs: []*ctypes.ResultTx{
				{Tx: []byte("invalid tx")},
			}},
			expectedError:       nil,
			expectedLogOutput:   "during refunding decoding tx () failed: decoding transaction () result failed: expected 2 wire type, got 1: tx parse error\r\nfailed to mint: during mint received amount without gas (7994995000000000000) is smaller than price (8000000000000000000)",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Sub(sdk.NewIntFromUint64(5005000000000000))))),
			},
		},
		{
			name: "ShouldHaveSingleMsgInRefundTx",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),
			sentBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "refund bank send tx should contain exactly one message but instead it contains 2\r\nfailed to mint: during mint received amount without gas (7994995000000000000) is smaller than price (8000000000000000000)",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Sub(sdk.NewIntFromUint64(5005000000000000))))),
			},
		},
		{
			name: "ShouldHaveSingleMsgInRefundTxWhichlShouldBeMsgSend",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),
			sentBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgMultiSend(nil, nil),
				},
			}, []string{
				"",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "refund bank send not valid bank send\r\nfailed to mint: during mint received amount without gas (7994995000000000000) is smaller than price (8000000000000000000)",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Sub(sdk.NewIntFromUint64(5005000000000000))))),
			},
		},
		{
			name: "ShouldHaveWalletAsRefundSender",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),
			sentBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "refund bank send from expected cudos1a326k254fukx9jlp0h3fwcr2ymjgludzum67dv but actual is cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg\r\nfailed to mint: during mint received amount without gas (7994995000000000000) is smaller than price (8000000000000000000)",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Sub(sdk.NewIntFromUint64(5005000000000000))))),
			},
		},
		{
			name: "ShouldHaveTheBankSendSenderAsRefundReceiver",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),

			sentBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(wallet, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "refund bank send to expected cudos1vz78ezuzskf9fgnjkmeks75xum49hug6l2wgeg but actual is cudos1a326k254fukx9jlp0h3fwcr2ymjgludzum67dv\r\nfailed to mint: during mint received amount without gas (7994995000000000000) is smaller than price (8000000000000000000)",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Sub(sdk.NewIntFromUint64(5005000000000000))))),
			},
		},
		{
			name: "ShouldSkipRefundIfAlreadyRefunded",
			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, "refundhash#1"),

			sentBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"refundhash#1",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "already refunded 726566756E64686173682331\r\nfailed to mint: during mint received amount without gas (7994995000000000000) is smaller than price (8000000000000000000)",
			expectedOutputMemos: []string{},
			expectedOutputMsgs:  []sdk.Msg{},
		},
		{
			name: "ShouldRefundMintRequestForNftWithNonApprovedStatus",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Add(sdk.NewIntFromUint64(5005000000000000))))),
				},
			}, []string{
				"{\"uid\":\"nftuid#2\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "failed to mint: nft (nftuid#2) has invalid status (rejected)",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
			},
		},
		{
			name: "ShouldSuccessfullyRefundIfCoinsLessThanPriceWithGas",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "failed to mint: during mint received amount without gas (7994995000000000000) is smaller than price (8000000000000000000)",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Sub(sdk.NewIntFromUint64(5005000000000000))))),
			},
		},
		{
			name: "ShouldSuccessfullyRefundIfNftDataNotFound",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000)))),
				},
			}, []string{
				"{\"uid\":\"notfoundnftuid\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "failed to mint: nft (notfoundnftuid) was not found",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				banktypes.NewMsgSend(wallet, buyer1, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Sub(sdk.NewIntFromUint64(5005000000000000))))),
			},
		},
		{
			name: "ShouldSuccessfullyMintNftEvenIfMarkingFails",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Add(sdk.NewIntFromUint64(5005000000000000))))),
				},
			}, []string{
				"{\"uid\":\"nftuid#3\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "failed marking nft (nftuid#3) as minted: not found",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				marketplacetypes.NewMsgMintNft(wallet.String(), "testdenom", buyer1.String(), "test nft name", "test nft uri", "test nft data", "nftuid#3",
					sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000))),
			},
		},
		{
			name: "ShouldSuccessfullyMintNft",

			receivedBankSendTxs: buildTestResultTxSearch(t, [][]sdk.Msg{
				{
					banktypes.NewMsgSend(buyer1, wallet, sdk.NewCoins(sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000).Add(sdk.NewIntFromUint64(5005000000000000))))),
				},
			}, []string{
				"{\"uid\":\"nftuid#1\"}",
			}, encodingConfig, ""),

			expectedError:       nil,
			expectedLogOutput:   "",
			expectedOutputMemos: []string{""},
			expectedOutputMsgs: []sdk.Msg{
				marketplacetypes.NewMsgMintNft(wallet.String(), "testdenom", buyer1.String(), "test nft name", "test nft uri", "test nft data", "nftuid#1",
					sdk.NewCoin("acudos", sdk.NewIntFromUint64(8000000000000000000))),
			},
		},
	}
}

type testCase struct {
	name                string
	receivedBankSendTxs *ctypes.ResultTxSearch
	sentBankSendTxs     *ctypes.ResultTxSearch
	mintTxs             *ctypes.ResultTxSearch
	expectedError       error
	expectedLogOutput   string
	expectedOutputMemos []string
	expectedOutputMsgs  []sdk.Msg
}

func buildTestResultTxSearch(t *testing.T, msgs [][]sdk.Msg, memos []string, encodingConfig *params.EncodingConfig, txHash string) *ctypes.ResultTxSearch {
	require.Len(t, msgs, len(memos))

	resultTxSearch := ctypes.ResultTxSearch{}

	for i := range msgs {
		resultTx := &ctypes.ResultTx{
			Hash:   bytes.HexBytes(txHash),
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
