package relayminter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	marketplacetypes "github.com/CudoVentures/cudos-node/x/marketplace/types"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	queryacc "github.com/CudoVentures/cudos-ondemand-minting-service/internal/query/account"
	relaytx "github.com/CudoVentures/cudos-ondemand-minting-service/internal/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
	ggrpc "google.golang.org/grpc"
)

func NewRelayMinter(logger relayLogger, encodingConfig *params.EncodingConfig, cfg config.Config, stateStorage stateStorage,
	nftDataClient nftDataClient, privKey *secp256k1.PrivKey, grpcConnector grpcConnector, rpcConnector rpcConnector, txCoder txCoder) *relayMinter {
	return &relayMinter{
		encodingConfig: encodingConfig,
		config:         cfg,
		stateStorage:   stateStorage,
		privKey:        privKey,
		nftDataClient:  nftDataClient,
		logger:         logger,
		walletAddress:  sdk.AccAddress(privKey.PubKey().Address()),
		grpcConnector:  grpcConnector,
		rpcConnector:   rpcConnector,
		txCoder:        txCoder,
	}
}

func (rm *relayMinter) Start(ctx context.Context) {
	rm.logger.Info("start relayer")

	retries := 0

	retry := func(err error) {
		rm.logger.Error(fmt.Errorf("relaying failed on retry %d of %d: %v", retries, rm.config.MaxRetries, err))

		ticker := time.NewTicker(rm.config.RetryInterval)

		select {
		case <-ticker.C:
		case <-ctx.Done():
		}

		retries += 1
	}

	for ctx.Err() == nil && retries < rm.config.MaxRetries {
		grpcConn, err := rm.grpcConnector.MakeGRPCClient(rm.config.ChainGRPC)
		if err != nil {
			retry(fmt.Errorf("dialing GRPC url (%s) failed: %s", rm.config.ChainGRPC, err))
			continue
		}
		defer grpcConn.Close()

		node, err := rm.rpcConnector.MakeRPCClient(rm.config.ChainRPC)
		if err != nil {
			retry(fmt.Errorf("connecting (%s) failed: %s", rm.config.ChainRPC, err))
			continue
		}
		defer node.Stop()

		rm.txSender = relaytx.NewTxSender(
			txtypes.NewServiceClient(grpcConn),
			queryacc.NewAccountInfoClient(grpcConn, rm.encodingConfig),
			rm.encodingConfig,
			rm.privKey,
			rm.config.ChainID,
			rm.config.PaymentDenom,
			gasPrice, gasAdjustment,
			relaytx.NewTxSigner(rm.encodingConfig, rm.privKey),
		)
		rm.txQuerier = relaytx.NewTxQuerier(node)

		rm.logger.Info("starting relayer loop")
		err = rm.startRelaying(ctx)

		if err == contextDone {
			return
		}

		retry(err)
	}

	rm.logger.Info("end relayer")
}

func (rm *relayMinter) startRelaying(ctx context.Context) error {
	ticker := time.NewTicker(rm.config.RelayInterval)

	for {
		select {
		case <-ticker.C:
			if err := rm.relay(ctx); err != nil {
				return err
			}
			ticker = time.NewTicker(rm.config.RelayInterval)
		case <-ctx.Done():
			return contextDone
		}
	}
}

// Get bank transfers to our address with height > s.Height
// Start iterating them
//
//			Check if given NFT UID is minted onchain, if true, then refund
//	     		Before refunding make sure you didn't refunded already by checking for refunds for this tx hash
//
// If mint fails for some reason, refund the user following the same refund checks as above
func (rm *relayMinter) relay(ctx context.Context) error {
	rm.logger.Info("relay tick")
	s, err := rm.stateStorage.GetState()
	if err != nil {
		return err
	}

	rm.logger.Infof("check events after %d of wallet %s", s.Height, rm.walletAddress.String())
	results, err := rm.txQuerier.Query(ctx, fmt.Sprintf("tx.height>%d AND transfer.recipient='%s'", s.Height, rm.walletAddress.String()))
	if err != nil {
		return err
	}

	if results == nil || len(results.Txs) == 0 {
		rm.logger.Info("there is nothing to process")
		return nil
	}

	rm.logger.Infof("successfully got %d events", len(results.Txs))
	sort.Slice(results.Txs, func(i, j int) bool {
		return results.Txs[i].Height < results.Txs[j].Height
	})

	for _, result := range results.Txs {
		incomingPaymentTxHash := result.Hash.String()
		sendInfo, err := rm.getReceivedBankSendInfo(result)
		if err != nil {
			rm.logger.Warnf("getting received bank send info for tx(%s) failed: %s", incomingPaymentTxHash, err)
			continue
		}
		rm.logger.Infof("processing incomingPaymentTxHash(%s) with payment(%s)", incomingPaymentTxHash, sendInfo.String())

		isMintedNft, err := rm.isMintedNft(ctx, sendInfo.Memo.UID)
		if err != nil {
			return err
		}

		if isMintedNft {
			if err := rm.refund(ctx, sendInfo.Memo.UID, incomingPaymentTxHash, sendInfo.FromAddress, sendInfo.Amount); err != nil {
				return fmt.Errorf("%s, failed to refund as it was already minted", err)
			}

			continue
		}

		nftData, err := rm.nftDataClient.GetNFTData(ctx, sendInfo.Memo.UID)
		if err != nil {
			return err
		}
		rm.logger.Infof("NFT Data(%s)", nftData.String())

		if errMint := rm.mint(ctx, incomingPaymentTxHash, sendInfo.Memo.UID, sendInfo.Memo.RecipientAddress, nftData, sendInfo.Amount); errMint != nil {
			errMint = fmt.Errorf("failed to mint: %s", errMint)
			rm.logger.Warnf("minting of NFT(%s) failed from address(%s) with tx incomingPaymentTxHash(%s) with error[%v]", sendInfo.Memo.UID, sendInfo.FromAddress, incomingPaymentTxHash, errMint)
			if errRefund := rm.refund(ctx, sendInfo.Memo.UID, incomingPaymentTxHash, sendInfo.FromAddress, sendInfo.Amount); errRefund != nil {
				return fmt.Errorf("%s, failed to refund after unsuccessfull minting: %s", errMint, errRefund)
			}
		}
	}

	// Update the height in state with the latest one from results because there will be no txs with lower height since blocks are finalized

	s.Height = results.Txs[len(results.Txs)-1].Height

	rm.logger.Info(fmt.Sprintf("update state to %d", s.Height))
	return rm.stateStorage.UpdateState(s)
}

func (rm *relayMinter) mint(ctx context.Context, incomingPaymentTxHash string, uid, recipient string, nftData model.NFTData, amount sdk.Coin) error {
	emptyNftData := model.NFTData{}

	if nftData.PriceValidUntil < time.Now().UnixMilli() {
		return fmt.Errorf("NftPrice valid time expired. Not minting it")
	}

	if nftData == emptyNftData {
		return fmt.Errorf("nft (%s) was not found", uid)
	}

	if nftData.Status != model.QueuedNFTStatus {
		return fmt.Errorf("nft (%s) has invalid status (%s)", uid, nftData.Status)
	}

	msgMintNft := marketplacetypes.NewMsgMintNft(rm.walletAddress.String(), nftData.DenomID, recipient, nftData.Name, nftData.Uri, nftData.Data, uid, sdk.NewCoin("acudos", nftData.Price))
	gasResult, err := rm.txSender.EstimateGas(ctx, []sdk.Msg{msgMintNft}, "")
	if err != nil {
		return err
	}

	gas := sdk.NewIntFromUint64(gasResult.GasLimit).Mul(sdk.NewIntFromUint64(gasPrice))
	if gas.GT(amount.Amount) {
		return fmt.Errorf("during mint received amount (%d) is smaller than the gas (%d)", amount.Amount.Uint64(), gas.Uint64())
	}

	amountWithoutGas := amount.Amount.Sub(gas)
	if amountWithoutGas.LT(nftData.Price) {
		return fmt.Errorf("during mint received amount without gas (%d) is smaller than price (%d)", amountWithoutGas.Uint64(), nftData.Price.Uint64())
	}

	txHash, err := rm.txSender.SendTx(ctx, []sdk.Msg{msgMintNft}, incomingPaymentTxHash, gasResult)
	if err != nil {
		return err
	}

	rm.logger.Infof("success mint tx %s", txHash)
	if err := rm.nftDataClient.MarkMintedNFT(ctx, txHash, uid); err != nil {
		rm.logger.Error(fmt.Errorf("failed marking nft (%s) as minted: %s", uid, err))
	}

	return nil
}

// Refund only if not refunded already by checking onchain
// and the money that user sent are enough to cover the gas fees, otherwise skip
func (rm *relayMinter) refund(ctx context.Context, uid, incomingPaymentTxHash, refundReceiver string, amount sdk.Coin) error {
	isMintingTransaction, err := rm.isMintingTransaction(ctx, uid, incomingPaymentTxHash)
	if err != nil {
		return err
	}

	if isMintingTransaction {
		return nil
	}

	isRefunded, err := rm.isRefunded(ctx, incomingPaymentTxHash, refundReceiver)
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
	gasResult, err := rm.txSender.EstimateGas(ctx, []sdk.Msg{msgSend}, incomingPaymentTxHash)
	if err != nil {
		return err
	}

	amountWithoutGas := amount.Amount.Sub(sdk.NewIntFromUint64(gasResult.GasLimit).Mul(sdk.NewIntFromUint64(gasPrice)))
	// We want to have some min refund amount to prevent DoS
	if amountWithoutGas.LT(sdk.NewIntFromUint64(minRefundAmount)) {
		rm.logger.Error(fmt.Errorf("during refund received amount without gas (%d) is smaller than minimum refund amount (%d)", amountWithoutGas.Int64(), minRefundAmount))
		return nil
	}

	msgSend = banktypes.NewMsgSend(walletAddress, refundAddress, sdk.NewCoins(sdk.NewCoin(rm.config.PaymentDenom, amountWithoutGas)))
	refundTxHash, err := rm.txSender.SendTx(ctx, []sdk.Msg{msgSend}, incomingPaymentTxHash, gasResult)
	if err == nil {
		rm.logger.Info(fmt.Sprintf("successfull refund incomingPaymentTxHash(%s) to address(%s) with refund tx hash(%s)", incomingPaymentTxHash, refundReceiver, refundTxHash))
	}

	return err
}

func (rm *relayMinter) isMintedNft(ctx context.Context, uid string) (bool, error) {
	rm.logger.Infof("checking whether NFT(%s) is minted", uid)
	results, err := rm.queryNftMintTransaction(ctx, uid, "minted")
	if err != nil {
		return false, err
	}

	if len(results) > 0 {
		rm.logger.Infof("Minted NFT(%s): true [%s]", uid, results[0].Hash)
		return true, nil
	}

	rm.logger.Infof("Minted NFT(%s): false", uid)
	return false, nil
}

// func (rm *relayMinter) isMintedNft(ctx context.Context, uid string) (bool, error) {
// 	rm.logger.Infof("checking whether NFT(%s) is minted", uid)
// 	results, err := rm.txQuerier.Query(ctx, fmt.Sprintf("marketplace_mint_nft.uid='%s'", uid))
// 	if err != nil {
// 		return false, err
// 	}

// 	if results != nil && len(results.Txs) > 0 {
// 		for _, result := range results.Txs {
// 			tx, err := rm.decodeTx(result)
// 			if err != nil {
// 				rm.logger.Warnf("during check if minted, decoding tx (%s) failed: %s", result.Hash.String(), err)
// 				continue
// 			}

// 			msgs := tx.GetMsgs()
// 			if len(msgs) != 1 {
// 				rm.logger.Warnf("during check if minted for tx(%s), expected one message but got %d", result.Hash.String(), len(msgs))
// 				continue
// 			}

// 			mintMsg, ok := msgs[0].(*marketplacetypes.MsgMintNft)
// 			if !ok {
// 				rm.logger.Warnf("during check if minted for tx(%s), message was not mint msg", result.Hash.String())
// 				continue
// 			}

// 			// TO DO: Why we are checking the creator?
// 			if mintMsg.Creator != rm.walletAddress.String() {
// 				rm.logger.Warnf("during check if minted for tx(%s), creator (%s) of the mint msg is not equal to wallet (%s)", result.Hash.String(), mintMsg.Creator, rm.walletAddress.String())
// 				continue
// 			}

// 			if mintMsg.Uid == uid {
// 				rm.logger.Infof("Minted NFT(%s): true [%s]", uid, result.Hash.String())
// 				return true, nil
// 			}
// 		}
// 	}

// 	rm.logger.Infof("Minted NFT(%s): false", uid)
// 	return false, nil
// }

func (rm *relayMinter) isMintingTransaction(ctx context.Context, uid, incomingPaymentTxHash string) (bool, error) {
	rm.logger.Infof("checking whether %s is minting transaction", incomingPaymentTxHash)
	results, err := rm.queryNftMintTransaction(ctx, uid, "minting transaction")
	if err != nil {
		return false, err
	}

	for _, result := range results {
		tx := result.TxWithMemo
		if tx.GetMemo() == incomingPaymentTxHash {
			rm.logger.Infof("%s is minting tx: true [%s]", incomingPaymentTxHash, result.Hash)
			return true, nil
		}
	}

	rm.logger.Infof("%s is minting tx: false", incomingPaymentTxHash)
	return false, nil
}

// func (rm *relayMinter) isMintingTransaction(ctx context.Context, uid, incomingPaymentTxHash string) (bool, error) {
// 	rm.logger.Infof("checking whether %s is minting transaction", incomingPaymentTxHash)
// 	results, err := rm.txQuerier.Query(ctx, fmt.Sprintf("marketplace_mint_nft.uid='%s'", uid))
// 	if err != nil {
// 		return false, err
// 	}

// 	if results != nil && len(results.Txs) > 0 {
// 		for _, result := range results.Txs {
// 			tx, err := rm.decodeTx(result)
// 			if err != nil {
// 				rm.logger.Warnf("during check if minting transaction, decoding tx (%s) failed: %s", result.Hash.String(), err)
// 				continue
// 			}

// 			msgs := tx.GetMsgs()
// 			if len(msgs) != 1 {
// 				rm.logger.Warnf("during check if minting transaction for tx(%s), expected one message but got %d", result.Hash.String(), len(msgs))
// 				continue
// 			}

// 			mintMsg, ok := msgs[0].(*marketplacetypes.MsgMintNft)
// 			if !ok {
// 				rm.logger.Warnf("during check if minting transaction for tx(%s), message was not mint msg", result.Hash.String())
// 				continue
// 			}

// 			// TO DO: Why we are checking the creator?
// 			if mintMsg.Creator != rm.walletAddress.String() {
// 				rm.logger.Warnf("during check if minting transaction for tx(%s), creator (%s) of the mint msg is not equal to wallet (%s)", result.Hash.String(), mintMsg.Creator, rm.walletAddress.String())
// 				continue
// 			}

// 			if mintMsg.Uid != uid {
// 				rm.logger.Warnf("during check if minting transaction for tx(%s), msg.uid (%s) != nft uid(%s)", result.Hash.String(), mintMsg.Uid, uid)
// 				continue
// 			}

// 			if tx.GetMemo() == incomingPaymentTxHash {
// 				rm.logger.Infof("%s is minting tx: true [%s]", incomingPaymentTxHash, result.Hash.String())
// 				return true, nil
// 			}
// 		}
// 	}

// 	rm.logger.Infof("%s is minting tx: false", incomingPaymentTxHash)
// 	return false, nil
// }

func (rm *relayMinter) isRefunded(ctx context.Context, incomingPaymentTxHash, refundReceiver string) (bool, error) {
	rm.logger.Infof("checking whether %s is refunded", incomingPaymentTxHash)
	results, err := rm.txQuerier.Query(ctx, fmt.Sprintf("transfer.sender='%s' AND transfer.recipient='%s'", rm.walletAddress, refundReceiver))
	if err != nil {
		return false, err
	}

	if results != nil && len(results.Txs) > 0 {
		// Errors won't propagandate to the callers, because we don't wanna retry on errors related to parsing the tx
		// The only case we would have errors here is if some attacker manages to generate tx that is returned by above query
		for _, result := range results.Txs {
			txWithMemo, err := rm.decodeTx(result)
			if err != nil {
				rm.logger.Warnf("during check if refunded, decoding tx (%s) failed: %s", result.Hash.String(), err)
				continue
			}

			msgs := txWithMemo.GetMsgs()
			if len(msgs) != 1 {
				rm.logger.Warnf("during check if refunded for tx(%s), refund bank send should contain exactly one message but instead it contains %d", result.Hash.String(), len(msgs))
				continue
			}

			bankSendMsg, ok := msgs[0].(*banktypes.MsgSend)
			if !ok {
				rm.logger.Warnf("during check if refunded for tx(%s), refund bank send not valid bank send", result.Hash.String())
				continue
			}

			if bankSendMsg.FromAddress != rm.walletAddress.String() {
				rm.logger.Warnf("during check if refunded for tx(%s), refund bank send from expected %s but actual is %s", result.Hash.String(), rm.walletAddress.String(), bankSendMsg.FromAddress)
				continue
			}

			if bankSendMsg.ToAddress != refundReceiver {
				rm.logger.Warnf("during check if refunded for tx(%s), refund bank send to expected %s but actual is %s", result.Hash.String(), refundReceiver, bankSendMsg.ToAddress)
				continue
			}

			if txWithMemo.GetMemo() == incomingPaymentTxHash {
				rm.logger.Infof("%s refunded: true [%s]", incomingPaymentTxHash, result.Hash.String())
				return true, nil
			}
		}
	}

	rm.logger.Infof("%s refunded: false", incomingPaymentTxHash)
	return false, nil
}

func (rm *relayMinter) queryNftMintTransaction(ctx context.Context, uid, logInfo string) ([]*decodedTxWithMemo, error) {
	resultingArray := make([](*decodedTxWithMemo), 0)

	results, err := rm.txQuerier.Query(ctx, fmt.Sprintf("marketplace_mint_nft.uid='%s'", uid))
	if err != nil {
		return resultingArray, err
	}

	if results != nil && len(results.Txs) > 0 {
		for _, result := range results.Txs {
			tx, err := rm.decodeTx(result)
			if err != nil {
				rm.logger.Warnf("during check if %s, decoding tx (%s) failed: %s", logInfo, result.Hash.String(), err)
				continue
			}

			msgs := tx.GetMsgs()
			if len(msgs) != 1 {
				rm.logger.Warnf("during check if %s for tx(%s), expected one message but got %d", logInfo, result.Hash.String(), len(msgs))
				continue
			}

			mintMsg, ok := msgs[0].(*marketplacetypes.MsgMintNft)
			if !ok {
				rm.logger.Warnf("during check if %s for tx(%s), message was not mint msg", logInfo, result.Hash.String())
				continue
			}

			// TO DO: Why we are checking the creator?
			if mintMsg.Creator != rm.walletAddress.String() {
				rm.logger.Warnf("during check if %s for tx(%s), creator (%s) of the mint msg is not equal to wallet (%s)", logInfo, result.Hash.String(), mintMsg.Creator, rm.walletAddress.String())
				continue
			}

			if mintMsg.Uid == uid {
				resultingArray = append(resultingArray, NewDecodedTxWithMemo(result.Hash.String(), tx))
			}
		}
	}

	return resultingArray, nil
}

func (rm *relayMinter) EstimateGas(ctx context.Context, msgs []sdk.Msg, memo string) (model.GasResult, error) {
	return rm.txSender.EstimateGas(ctx, msgs, memo)
}

func (rm *relayMinter) GetNFTData(ctx context.Context, uid string) (model.NFTData, error) {
	return rm.nftDataClient.GetNFTData(ctx, uid)
}

func (rm *relayMinter) decodeTx(resultTx *ctypes.ResultTx) (sdk.TxWithMemo, error) {
	tx, err := rm.txCoder.Decode(resultTx.Tx)
	if err != nil {
		return nil, fmt.Errorf("decoding transaction (%s) result failed: %s", resultTx.Hash.String(), err)
	}

	txWithMemo, ok := tx.(sdk.TxWithMemo)
	if !ok {
		return nil, fmt.Errorf("invalid transaction (%s) type, it does not have memo", resultTx.Hash.String())
	}

	return txWithMemo, nil
}

// If any error is returned here, it means that the transaction or message are invalid, so in the processing loop we skip this tx
func (rm *relayMinter) getReceivedBankSendInfo(resultTx *ctypes.ResultTx) (receivedBankSend, error) {
	txWithMemo, err := rm.decodeTx(resultTx)
	if err != nil {
		return receivedBankSend{}, fmt.Errorf("getting received bank info: %s", err)
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

	if bankSendMsg.ToAddress != rm.walletAddress.String() {
		return receivedBankSend{}, fmt.Errorf("bank send receiver (%s) is not the wallet (%s)", bankSendMsg.ToAddress, rm.walletAddress.String())
	}

	if len(bankSendMsg.Amount) != 1 {
		return receivedBankSend{}, fmt.Errorf("bank send should have single coin sent instead got %+v", bankSendMsg.Amount)
	}

	if bankSendMsg.Amount[0].Denom != rm.config.PaymentDenom {
		return receivedBankSend{}, fmt.Errorf("bank send invalid payment denom, expected %s but got %s", rm.config.PaymentDenom, bankSendMsg.Amount[0].Denom)
	}

	if memo.RecipientAddress == "" {
		memo.RecipientAddress = bankSendMsg.FromAddress
	}

	return receivedBankSend{
		Memo:        memo,
		FromAddress: bankSendMsg.FromAddress,
		ToAddress:   bankSendMsg.ToAddress,
		Amount:      bankSendMsg.Amount[0],
	}, nil
}

const (
	gasPrice        = uint64(5000000000000)
	gasAdjustment   = float64(1.3)
	minRefundAmount = 5000000000000000000
)

var contextDone = errors.New("context done")

type relayMinter struct {
	encodingConfig *params.EncodingConfig
	errored        chan error
	config         config.Config
	stateStorage   stateStorage
	privKey        *secp256k1.PrivKey
	walletAddress  sdk.AccAddress
	txSender       txSender
	txQuerier      txQuerier
	nftDataClient  nftDataClient
	logger         relayLogger
	grpcConnector  grpcConnector
	rpcConnector   rpcConnector
	txCoder        txCoder
}

type mintMemo struct {
	UID               string `json:"uuid"`
	RecipientAddress  string `json:"recipientAddress"`
	ContractPaymentId string `json:"contractPaymentId"`
}

func (t *mintMemo) String() string {
	return fmt.Sprintf("MintMemo { UID(%s) RecipientAddress(%s) ContractPaymentId(%s) }", t.UID, t.RecipientAddress, t.ContractPaymentId)
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

func (t *receivedBankSend) String() string {
	return fmt.Sprintf("ReceivedBankSend { Memo(%s) FromAddress(%s) ToAddress(%s) Amount(%s) }", t.Memo.String(), t.FromAddress, t.ToAddress, t.Amount.String())
}

type stateStorage interface {
	GetState() (model.State, error)
	UpdateState(state model.State) error
}

type txCoder interface {
	Decode(tx tmtypes.Tx) (sdk.Tx, error)
}

type txSender interface {
	EstimateGas(ctx context.Context, msgs []sdk.Msg, memo string) (model.GasResult, error)
	SendTx(ctx context.Context, msgs []sdk.Msg, memo string, gasResult model.GasResult) (string, error)
}

type txQuerier interface {
	Query(ctx context.Context, query string) (*ctypes.ResultTxSearch, error)
}

type nftDataClient interface {
	GetNFTData(ctx context.Context, uid string) (model.NFTData, error)
	MarkMintedNFT(ctx context.Context, txHash, uid string) error
}

type relayLogger interface {
	Error(err error)
	Info(msg string)
	Infof(format string, v ...interface{})
	Warn(msg string)
	Warnf(format string, v ...interface{})
}

type grpcConnector interface {
	MakeGRPCClient(url string) (*ggrpc.ClientConn, error)
}

type rpcConnector interface {
	MakeRPCClient(url string) (*rpchttp.HTTP, error)
}

type decodedTxWithMemo struct {
	Hash       string
	TxWithMemo sdk.TxWithMemo
}

func NewDecodedTxWithMemo(hash string, txWithMemo sdk.TxWithMemo) *decodedTxWithMemo {
	return &decodedTxWithMemo{
		Hash:       hash,
		TxWithMemo: txWithMemo,
	}
}
