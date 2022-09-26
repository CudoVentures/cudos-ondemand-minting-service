package relayminter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	marketplacetypes "github.com/CudoVentures/cudos-node/x/marketplace/types"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	queryacc "github.com/CudoVentures/cudos-ondemand-minting-service/internal/query/account"
	relaytx "github.com/CudoVentures/cudos-ondemand-minting-service/internal/tx"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/rs/zerolog/log"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc"
)

func NewRelayMinter(encodingConfig params.EncodingConfig, cfg config.Config, stateStorage stateStorage, nftDataClient nftDataClient, privKey *secp256k1.PrivKey) *relayMinter {
	return &relayMinter{
		encodingConfig: encodingConfig,
		errored:        make(chan error),
		config:         cfg,
		stateStorage:   stateStorage,
		privKey:        privKey,
		nftDataClient:  nftDataClient,
	}
}

func (rm *relayMinter) Start(ctx context.Context) error {

	defer close(rm.errored)

	rm.walletAddress = sdk.AccAddress(rm.privKey.PubKey().Address())

	retries := 0

	retry := func(err error) {
		log.Error().Err(fmt.Errorf("relaying failed: %v", err)).Send()

		time.Sleep(rm.config.RetryInterval)

		retries += 1
	}

	for retries < rm.config.MaxRetries {

		grpcConn, err := grpc.Dial(rm.config.Chain.GRPC, grpc.WithInsecure())
		defer grpcConn.Close()
		if err != nil {
			retry(fmt.Errorf("dialing GRPC url (%s) failed: %s", rm.config.Chain.GRPC, err))
			continue
		}

		rm.grpcConn = grpcConn

		node, err := client.NewClientFromNode(rm.config.Chain.RPC)
		if err != nil {
			retry(fmt.Errorf("connecting (%s) failed: %s", rm.config.Chain.RPC, err))
			continue
		}

		rm.node = node

		rm.txSender = relaytx.NewTxSender(grpcConn, queryacc.NewAccountInfoClient(grpcConn, rm.encodingConfig), rm.encodingConfig, rm.privKey, rm.config.Chain.ID, rm.config.PaymentDenom)

		go rm.startRelaying(ctx)

		retry(<-rm.errored)
	}

	return <-rm.errored
}

func (rm *relayMinter) startRelaying(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			rm.errored <- fmt.Errorf("panicked while relaying: %+v", r)
		}
	}()

	for {
		if err := rm.relay(ctx); err != nil {
			rm.errored <- err
			return
		}

		time.Sleep(rm.config.RelayInterval)
	}
}

// Get bank transfers to our address with height > s.Height
// Start iterating them
// 		Check if given NFT UID is minted onchain, if true, then refund
//      		Before refunding make sure you didn't refunded already by checking for refunds for this tx hash
// If mint fails for some reason, refund the user following the same refund checks as above
func (rm *relayMinter) relay(ctx context.Context) error {
	s, err := rm.stateStorage.GetState()
	if err != nil {
		return err
	}

	results, err := rm.getTxsByQuery(fmt.Sprintf("tx.height>%d AND transfer.recipient='%s'", s.Height, rm.walletAddress))
	if err != nil {
		return err
	}

	if results == nil || len(results.Txs) == 0 {
		return nil
	}

	// TODO: Verify that results is really sorted in ascending order by height

	for _, result := range results.Txs {
		sendInfo, err := rm.getReceivedBankSendInfo(result)
		if err != nil {
			continue
		}

		isMinted, err := rm.isMinted(sendInfo.Memo.UID)
		if err != nil {
			return err
		}

		if isMinted {
			if err := rm.refund(result.Hash.String(), sendInfo.FromAddress, sendInfo.Amount); err != nil {
				return err
			}

			continue
		}

		nftData, err := rm.nftDataClient.GetNFTData(sendInfo.Memo.UID)
		if err != nil {
			return err
		}

		if errMint := rm.mint(sendInfo.Memo.UID, sendInfo.FromAddress, nftData, sendInfo.Amount); errMint != nil {
			errMint = fmt.Errorf("failed to mint: %s", errMint)
			if errRefund := rm.refund(result.Hash.String(), sendInfo.FromAddress, sendInfo.Amount); errRefund != nil {
				return fmt.Errorf("%s, failed to refund: %s", errMint, errRefund)
			}
			return errMint
		}
	}

	// Update the height in state with the latest one from results because there will be no txs with lower height since blocks are finalized

	s.Height = results.Txs[len(results.Txs)-1].Height

	return rm.stateStorage.UpdateState(s)
}

func (rm *relayMinter) getTxsByQuery(query string) (*ctypes.ResultTxSearch, error) {
	txSearchCtx, cancelFunc := context.WithTimeout(context.Background(), txSearchTimeout)
	defer cancelFunc()

	// TODO: Do we need to do pagination? Will we get all results if we don't use pagination or there is some limit?
	results, err := rm.node.TxSearch(txSearchCtx, query, true, nil, nil, "asc")
	if err != nil {
		return nil, fmt.Errorf("tx search (%s) failed: %s", query, err)
	}

	return results, nil
}

func (rm *relayMinter) decodeTx(resultTx *ctypes.ResultTx) (sdk.TxWithMemo, error) {
	tx, err := rm.encodingConfig.TxConfig.TxDecoder()(resultTx.Tx)
	if err != nil {
		return nil, fmt.Errorf("decoding transaction (%s) result failed: %s", resultTx.Hash.String(), err)
	}

	txWithMemo, ok := tx.(sdk.TxWithMemo)
	if !ok {
		return nil, fmt.Errorf("invalid transaction (%s) type: %s", resultTx.Hash.String(), err)
	}

	return txWithMemo, nil
}

// If any error is returned here, it means that the transaction or message are invalid, so in the processing loop we skip this tx
func (rm *relayMinter) getReceivedBankSendInfo(resultTx *ctypes.ResultTx) (receivedBankSend, error) {
	txWithMemo, err := rm.decodeTx(resultTx)
	if err != nil {
		return receivedBankSend{}, err
	}

	var memo mintMemo
	memoStr := txWithMemo.GetMemo()
	if memoStr == "" {
		return receivedBankSend{}, fmt.Errorf("memo not set in transaction (%s)", resultTx.Hash.String())
	}

	if err := json.Unmarshal([]byte(memoStr), &memo); err != nil {
		return receivedBankSend{}, fmt.Errorf("unmarshaling memo (%s) failed: %s", memoStr, err)
	}

	if memo.UID == "" {
		return receivedBankSend{}, fmt.Errorf("empty memo UID in transaction (%s)", resultTx.Hash.String())
	}

	msgs := txWithMemo.GetMsgs()
	if len(msgs) != 1 {
		return receivedBankSend{}, fmt.Errorf("received bank send tx should contain exactly one message but instead it contains %d", len(msgs))
	}

	bankSendMsg, ok := msgs[0].(*banktypes.MsgSend)
	if !ok {
		return receivedBankSend{}, errors.New("not valid bank send")
	}

	if bankSendMsg.ToAddress != string(rm.walletAddress) {
		return receivedBankSend{}, fmt.Errorf("bank send receiver (%s) is not the wallet (%s)", bankSendMsg.ToAddress, rm.walletAddress)
	}

	if len(bankSendMsg.Amount) != 1 {
		return receivedBankSend{}, fmt.Errorf("bank send should have single coin sent instead got %+v", bankSendMsg.Amount)
	}

	if bankSendMsg.Amount[0].Denom != rm.config.PaymentDenom {
		return receivedBankSend{}, fmt.Errorf("bank send invalid payment denom, expected %s but got %s", rm.config.PaymentDenom, bankSendMsg.Amount[0].Denom)
	}

	return receivedBankSend{
		Memo:        memo,
		FromAddress: bankSendMsg.FromAddress,
		ToAddress:   bankSendMsg.ToAddress,
		Amount:      bankSendMsg.Amount[0],
	}, nil
}

func (rm *relayMinter) isMinted(uid string) (bool, error) {
	results, err := rm.getTxsByQuery(fmt.Sprintf("marketplace_mint_nft.uid='%s'", uid))
	if err != nil {
		return false, err
	}
	if results != nil && len(results.Txs) != 0 {
		return true, nil
	}
	return false, nil
}

func (rm *relayMinter) isRefunded(receiveTxHash, refundReceiver string) (bool, error) {
	results, err := rm.getTxsByQuery(fmt.Sprintf("transfer.sender='%s' AND transfer.recipient='%s'", rm.walletAddress, refundReceiver))
	if err != nil {
		return false, err
	}
	if results == nil || len(results.Txs) == 0 {
		return false, nil
	}

	// Errors won't propagandate to the callers, because we don't wanna retry on errors related to parsing the tx
	// The only case we would have errors here is if some attacker manages to generate tx that is returned by above query

	for _, result := range results.Txs {
		txWithMemo, err := rm.decodeTx(result)
		if err != nil {
			log.Error().Err(fmt.Errorf("decoding tx (%s) failed: %s", result.Hash, err)).Send()
			continue
		}

		msgs := txWithMemo.GetMsgs()
		if len(msgs) != 1 {
			log.Error().Err(fmt.Errorf("refund bank send tx should contain exactly one message but instead it contains %d", len(msgs))).Send()
			continue
		}

		bankSendMsg, ok := msgs[0].(*banktypes.MsgSend)
		if !ok {
			log.Error().Err(errors.New("not valid bank send"))
			continue
		}

		if bankSendMsg.FromAddress != rm.walletAddress.String() {
			log.Error().Err(fmt.Errorf("refund bank send from expected %s but actual is %s", rm.walletAddress.String(), bankSendMsg.FromAddress)).Send()
			continue
		}

		if bankSendMsg.ToAddress != refundReceiver {
			log.Error().Err(fmt.Errorf("refund bank send to expected %s but actual is %s", refundReceiver, bankSendMsg.ToAddress)).Send()
			continue
		}

		if txWithMemo.GetMemo() == receiveTxHash {
			return true, nil
		}
	}

	return false, nil
}

// Refund only if not refunded already by checking onchain
// and the money that user sent are enough to cover the gas fees, otherwise skip
func (rm *relayMinter) refund(txHash, refundReceiver string, amount sdk.Coin) error {
	isRefunded, err := rm.isRefunded(txHash, refundReceiver)
	if err != nil {
		return err
	}

	if isRefunded {
		return nil
	}

	walletAddress, err := sdk.AccAddressFromBech32(rm.walletAddress.String())
	if err != nil {
		return fmt.Errorf("invalid wallet address (%s) during refund: %s", rm.walletAddress, err)
	}

	refundAddress, err := sdk.AccAddressFromBech32(refundReceiver)
	if err != nil {
		return fmt.Errorf("invalid refund receiver address (%s) during refund: %s", refundReceiver, err)
	}

	msgSend := banktypes.NewMsgSend(walletAddress, refundAddress, sdk.NewCoins(amount))

	gasResult, err := rm.txSender.EstimateGas([]sdk.Msg{msgSend})
	if err != nil {
		return err
	}

	amountWithoutGas := amount.Amount.Sub(sdk.NewIntFromUint64(gasResult.GasLimit))

	// We want to have some min refund amount to prevent DoS
	if amountWithoutGas.LT(sdk.NewIntFromUint64(minRefundAmount)) {
		return fmt.Errorf("during refund received amount without gas (%d) is smaller than minimum refund amount (%d)",
			amountWithoutGas.Uint64(), minRefundAmount)
	}

	return rm.txSender.SendTx([]sdk.Msg{msgSend}, txHash)
}

func (rm *relayMinter) mint(uid, recipient string, nftData model.NFTData, amount sdk.Coin) error {
	if nftData.Status != model.ApprovedNFTStatus {
		return fmt.Errorf("nft (%s) has invalid status (%s)", uid, nftData.Status)
	}

	msgMintNft := marketplacetypes.NewMsgMintNft(rm.walletAddress.String(), nftData.DenomID,
		recipient, nftData.Name, nftData.Uri, nftData.Data, uid, nftData.Price)

	gasResult, err := rm.txSender.EstimateGas([]sdk.Msg{msgMintNft})
	if err != nil {
		return err
	}

	amountWithoutGas := amount.Amount.Sub(sdk.NewIntFromUint64(gasResult.GasLimit))

	if amountWithoutGas.LT(nftData.Price.Amount) {
		return fmt.Errorf("during mint received amount without gas (%d) is smaller than price (%d)",
			amountWithoutGas.Uint64(), nftData.Price.Amount.Uint64())
	}

	if err := rm.txSender.SendTx([]sdk.Msg{msgMintNft}, ""); err != nil {
		return err
	}

	if err := rm.nftDataClient.MarkMintedNFT(uid); err != nil {
		log.Error().Err(fmt.Errorf("failed marking nft (%s) as minted: %s", uid, err)).Send()
	}

	return nil
}

const txSearchTimeout = 10 * time.Second
const minRefundAmount = 5000000000000000000

type relayMinter struct {
	encodingConfig params.EncodingConfig
	errored        chan error
	config         config.Config
	stateStorage   stateStorage
	privKey        *secp256k1.PrivKey
	walletAddress  sdk.AccAddress
	node           *rpchttp.HTTP
	grpcConn       *grpc.ClientConn
	txSender       txSender
	nftDataClient  nftDataClient
}

type mintMemo struct {
	UID string `json:"uid"`
}

type refundMemo struct {
	TxHash string `json:"tx_hash"`
}

type receivedBankSend struct {
	Memo        mintMemo
	FromAddress string
	ToAddress   string
	Amount      sdk.Coin
}

type stateStorage interface {
	GetState() (model.State, error)
	UpdateState(state model.State) error
}

type txSender interface {
	EstimateGas(msgs []sdk.Msg) (model.GasResult, error)
	SendTx(msgs []sdk.Msg, memo string) error
}

type nftDataClient interface {
	GetNFTData(uid string) (model.NFTData, error)
	MarkMintedNFT(uid string) error
}
