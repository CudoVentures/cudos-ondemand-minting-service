package relayminter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"cosmossdk.io/simapp/params"
	marketplacetypes "github.com/CudoVentures/cudos-node/x/marketplace/types"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	queryacc "github.com/CudoVentures/cudos-ondemand-minting-service/internal/query/account"
	relaytx "github.com/CudoVentures/cudos-ondemand-minting-service/internal/tx"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ggrpc "google.golang.org/grpc"
)

// The RelayerMinter is responsible for cudos chain monitoring and minting of the NFTs to its owner.
func NewRelayMinter(logger relayLogger, encodingConfig *params.EncodingConfig, cfg config.Config, stateStorage stateStorage,
	nftDataClient nftDataClient, privKey *secp256k1.PrivKey, grpcConnector grpcConnector, rpcConnector rpcConnector, txCoder txCoder, emailService emailService) *relayMinter {
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
		retries:        0,
		emailService:   emailService,
	}
}

// Starting a routine that is relaying the incoming transactions.
// The relayer is retried in case of an error and nothing furcher is processes unless the error is resolved. A service is email is send in a case of an error as well.
// The error counter is reset in case of successful relay.
// The relayer exists once it reach max retires defined in the cfg.
func (rm *relayMinter) Start(ctx context.Context) {
	rm.logger.Info("starting relayer")

	retry := func(err error) {
		errorMessage := fmt.Sprintf("relaying failed on retry %d of %d: %v", rm.retries, rm.config.MaxRetries, err)
		rm.logger.Error(errors.New(errorMessage))
		rm.emailService.SendEmail(errorMessage)

		ticker := time.NewTicker(rm.config.RetryInterval)

		select {
		case <-ticker.C:
		case <-ctx.Done():
		}

		rm.retries += 1
	}

	for ctx.Err() == nil && rm.retries < rm.config.MaxRetries {
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
			rm.logger.Error(err)
			return
		}

		retry(err)
	}

	rm.logger.Info("stopping relayer")
}

// Creating a ticker. It invokes the relayer function once per tick.
func (rm *relayMinter) startRelaying(ctx context.Context) error {
	ticker := time.NewTicker(rm.config.RelayInterval)

	for {
		select {
		case <-ticker.C:
			if err := rm.relay(ctx); err != nil {
				return err
			}
			rm.logger.Info("successfull relay. resetting retries")
			rm.retries = 0
			ticker = time.NewTicker(rm.config.RelayInterval)
		case <-ctx.Done():
			return contextDone
		}
	}
}

// Processing a single relay tick.
//
// Getting the transactions from last know processed block stored in the state up to latest block.
// Processing transactions one by one. If there is an error in some of the steps in the following the algorithm then the relay stops:
//
// 1. Find the corresponding information in the memo of a transaction. If no such information is available then no futher processing is required and moves to next transaction.
//
// 2. Checking if the transaction is a "minting transaction", which means whether this transaction resulted in a minted nft.
// If so then no futher processsing is required because the NFT that is supposed to be minted by this transaction has already been minted. Proceed with next transaction.
//
// 4. Checking if the transaction has already been refunded.
// If so then no further processing is required.
//
// 5. Getting NFT's information from the AuraPool using the informtion in transaction's memo. The AuraPool make all relevant checks and returns the correct NFT's data.
// If invalid data is returns from the AuraPool then some of the criterias are not met and the transaction is refunded. After the refund no further processing is required.
// From that point onwards only the information from AuraPool must be used
//
// 6. Checking if the NFT has ready been minted. If it is then the transaction is refunded. After the refund no further processing is required.
//
// 7. Trying to mint the NFT and refunding the transaction if minting is not successful.
func (rm *relayMinter) relay(ctx context.Context) error {
	rm.logger.Info("relay tick")
	s, err := rm.stateStorage.GetState()
	if err != nil {
		return err
	}

	rm.logger.Infof("check events after %d of wallet %s", s.Height, rm.walletAddress.String())
	// results, err := rm.txQuerier.Query(ctx, fmt.Sprintf("tx.height>%d AND transfer.recipient='%s'", s.Height, rm.walletAddress.String()))
	results, err := rm.txQuerier.QueryLegacy(ctx, []*relaytx.TxQuerierLegacyParams{
		{Key: "transfer.recipient", Value: rm.walletAddress.String()},
	}, s.Height+1)
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

	for i, result := range results.Txs {
		incomingPaymentTxHash := result.Hash.String()
		incomingPaymentTxHeight := result.Height
		sendInfo, err := rm.getReceivedBankSendInfo(result)
		if err != nil {
			rm.logger.Warnf("getting received bank send info for tx(%s) failed: %s", incomingPaymentTxHash, err)
			continue
		}
		rm.logger.Infof("%d: processing incomingPaymentTxHash(%s) at height(%d) with payment(%s)", i+1, incomingPaymentTxHash, result.Height, sendInfo.String())

		isMintingTransaction, err := rm.isMintingTransaction(ctx, sendInfo.Memo.RecipientAddress, incomingPaymentTxHash, incomingPaymentTxHeight)
		if err != nil {
			return err
		}

		if isMintingTransaction {
			rm.logger.Infof("transaction(%s) has already been successfully processed and it results to a minted nft to a buyer (%s)", incomingPaymentTxHash, sendInfo.Memo.RecipientAddress)
			continue
		}

		isRefunded, err := rm.isRefunded(ctx, incomingPaymentTxHash, incomingPaymentTxHeight, sendInfo.FromAddress)
		if err != nil {
			return err
		}

		if isRefunded {
			rm.logger.Infof("transaction(%s) has already been refunded to buyer(%s)", incomingPaymentTxHash, sendInfo.FromAddress)
			continue
		}

		// 300k gas * default gas price = 1.5CUDOS
		onCudos, _ := sdk.NewIntFromString("1500000000000000000")
		nftData, err := rm.GetNFTData(ctx, rm.config, sendInfo.Memo.UID, sendInfo.Memo.RecipientAddress, sendInfo.Amount.Sub(sdk.NewCoin("acudos", onCudos)))
		if err != nil {
			return err
		}
		rm.logger.Infof("NFT Data(%s)", nftData.String())

		isMintedNft, err := rm.isMintedNft(ctx, nftData.Id, incomingPaymentTxHeight)
		if err != nil {
			return err
		}

		if isMintedNft {
			if err := rm.refund(ctx, incomingPaymentTxHash, sendInfo.FromAddress, sendInfo.Amount); err != nil {
				return fmt.Errorf("%s, failed to refund as it was already minted", err)
			}

			continue
		}

		if errMint := rm.mint(ctx, incomingPaymentTxHash, nftData.Id, sendInfo.Memo.RecipientAddress, nftData, sendInfo.Amount); errMint != nil {
			errMint = fmt.Errorf("failed to mint: %s", errMint)
			rm.logger.Warnf("minting of NFT(%s) failed from address(%s) with tx incomingPaymentTxHash(%s) with error[%v]", nftData.Id, sendInfo.FromAddress, incomingPaymentTxHash, errMint)
			if errRefund := rm.refund(ctx, incomingPaymentTxHash, sendInfo.FromAddress, sendInfo.Amount); errRefund != nil {
				return fmt.Errorf("%s, failed to refund after unsuccessful minting: %s", errMint, errRefund)
			}
		}
	}

	// Update the height in state with the latest one from results because there will be no txs with lower height since blocks are finalized
	s.Height = results.Txs[len(results.Txs)-1].Height

	rm.logger.Info(fmt.Sprintf("update state to %d", s.Height))
	return rm.stateStorage.UpdateState(s)
}

// Mints the NFT
// If nft data received by the AuraPool is empty then return an error which will lead to a refund.
// The hash of incoming transaction is added as memo of the mint transaction
func (rm *relayMinter) mint(ctx context.Context, incomingPaymentTxHash string, uid, recipient string, nftData model.NFTData, amount sdk.Coin) error {
	emptyNftData := model.NFTData{}

	if nftData == emptyNftData {
		return fmt.Errorf("nft (%s) was not found", uid)
	}

	// this check is in AuraPool, but it can stay here just in case
	if nftData.PriceValidUntil < time.Now().UnixMilli() {
		return fmt.Errorf("NftPrice valid time expired. Not minting it")
	}

	// this check is in AuraPool, but it can stay here just in case
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
		return fmt.Errorf("during mint received amount (%s) is smaller than the gas (%s)", amount.Amount.String(), gas.String())
	}

	amountWithoutGas := amount.Amount.Sub(gas)
	if amountWithoutGas.LT(nftData.Price) {
		return fmt.Errorf("during mint received amount without gas (%s) is smaller than price (%s)", amountWithoutGas.String(), nftData.Price.String())
	}

	txHash, err := rm.txSender.SendTx(ctx, []sdk.Msg{msgMintNft}, incomingPaymentTxHash, gasResult)
	if err != nil {
		return err
	}

	rm.logger.Infof("success mint tx %s", txHash)
	return nil
}

// Refunds the user.
// The refunded amount is equal to incoming funds - refund transaction costs. This is so in order not to prevent draining of service's wallet funds.
// The hash of incoming transaction is added as memo of the refund transaction
func (rm *relayMinter) refund(ctx context.Context, incomingPaymentTxHash, refundReceiver string, amount sdk.Coin) error {
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
	if err != nil {
		return err
	}

	rm.logger.Info(fmt.Sprintf("successfull refund incomingPaymentTxHash(%s) to address(%s) with refund tx hash(%s)", incomingPaymentTxHash, refundReceiver, refundTxHash))
	return nil
}

// Checking if an NFT has already been minted.
// The checking is done by fetching all transactions by NFT's id emited by marketplace module.
// If such transaction exists then the NFT has already been minted
func (rm *relayMinter) isMintedNft(ctx context.Context, uid string, incomingPaymentTxHashHeight int64) (bool, error) {
	rm.logger.Infof("checking whether NFT(%s) is minted", uid)
	if uid == "" {
		return false, nil
	}

	results, err := rm.queryNftMintTransactionByUid(ctx, uid, incomingPaymentTxHashHeight, "minted")
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

// Checking whether an incoming transaction resulted in minted NFT.
// The checking is done by fetching all transactions by current buyer emited by marketplace module.
// If there is a transaction with memo = incoming transaction's hash then it means that the incoming transaction has already beed succesfully processed and there is a minted NFT as a result.
// This is TRUE because a mint transaction has a memo = incoming transaction's hash
func (rm *relayMinter) isMintingTransaction(ctx context.Context, buyerAddress, incomingPaymentTxHash string, incomingPaymentTxHashHeight int64) (bool, error) {
	rm.logger.Infof("checking whether %s is minting transaction", incomingPaymentTxHash)
	results, err := rm.queryNftMintTransactionByBuyer(ctx, buyerAddress, incomingPaymentTxHashHeight, "minting transaction")
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

// Checking whether an incoming transaction has already beed refunded.
// The checking is done by fetching all transactions from service's wallet to buyer's wallet.
// If there is a transaction with memo = incoming transaction's hash then it means that the incoming transaction has already beed refunded.
// This is TRUE because a refund transaction has a memo = incoming transaction's hash
func (rm *relayMinter) isRefunded(ctx context.Context, incomingPaymentTxHash string, incomingPaymentTxHeight int64, refundReceiver string) (bool, error) {
	rm.logger.Infof("checking whether %s is refunded", incomingPaymentTxHash)
	// results, err := rm.txQuerier.Query(ctx, fmt.Sprintf("tx.height>=%d AND transfer.sender='%s' AND transfer.recipient='%s'", incomingPaymentTxHeight, rm.walletAddress, refundReceiver))
	results, err := rm.txQuerier.QueryLegacy(ctx, []*relaytx.TxQuerierLegacyParams{
		{Key: "transfer.sender", Value: rm.walletAddress.String()},
		{Key: "transfer.recipient", Value: refundReceiver},
	}, incomingPaymentTxHeight)
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

// Fetching marketplace transactions from the chain by nft's id
func (rm *relayMinter) queryNftMintTransactionByUid(ctx context.Context, uid string, incomingPaymentTxHeight int64, logInfo string) ([]*decodedTxWithMemo, error) {
	resultingArray := make([](*decodedTxWithMemo), 0)

	// results, err := rm.txQuerier.Query(ctx, fmt.Sprintf("tx.height>=%d AND marketplace_mint_nft.uid='%s'", incomingPaymentTxHeight, uid))
	results, err := rm.txQuerier.QueryLegacy(ctx, []*relaytx.TxQuerierLegacyParams{
		{Key: "marketplace_mint_nft.uid", Value: uid},
	}, incomingPaymentTxHeight)
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

// Fetching marketplace transactions from the chain by buyer's address
func (rm *relayMinter) queryNftMintTransactionByBuyer(ctx context.Context, buyerAddress string, incomingPaymentTxHeight int64, logInfo string) ([]*decodedTxWithMemo, error) {
	resultingArray := make([](*decodedTxWithMemo), 0)

	// results, err := rm.txQuerier.Query(ctx, fmt.Sprintf("tx.height>=%d AND marketplace_mint_nft.buyer='%s'", incomingPaymentTxHeight, buyerAddress))
	results, err := rm.txQuerier.QueryLegacy(ctx, []*relaytx.TxQuerierLegacyParams{
		{Key: "marketplace_mint_nft.buyer", Value: buyerAddress},
	}, incomingPaymentTxHeight)
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

			if mintMsg.Recipient == buyerAddress {
				resultingArray = append(resultingArray, NewDecodedTxWithMemo(result.Hash.String(), tx))
			}
		}
	}

	return resultingArray, nil
}

func (rm *relayMinter) EstimateGas(ctx context.Context, msgs []sdk.Msg, memo string) (model.GasResult, error) {
	return rm.txSender.EstimateGas(ctx, msgs, memo)
}

func (rm *relayMinter) GetNFTData(ctx context.Context, cfg config.Config, uid, recipientCudosAddress string, paidAmount sdk.Coin) (model.NFTData, error) {
	return rm.nftDataClient.GetNFTData(ctx, rm.config, uid, recipientCudosAddress, paidAmount)
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

// Parsing a transaction's memo.
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
	gasAdjustment   = float64(1.5)
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
	retries        int
	emailService   emailService
}

type mintMemo struct {
	UID               string `json:"uuid"`
	RecipientAddress  string `json:"recipientAddress"`
	ContractPaymentId string `json:"contractPaymentId"`
	EthTxHash         string `json:"ethTxHash"`
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
	// Query(ctx context.Context, query string) (*ctypes.ResultTxSearch, error)
	QueryLegacy(ctx context.Context, query []*relaytx.TxQuerierLegacyParams, heights ...int64) (*ctypes.ResultTxSearch, error)
}

type nftDataClient interface {
	GetNFTData(ctx context.Context, cfg config.Config, uid, recipientCudosAddress string, amountPaid sdk.Coin) (model.NFTData, error)
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

type emailService interface {
	SendEmail(content string)
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
