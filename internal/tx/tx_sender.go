package tx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

func NewTxSender(txClient txtypes.ServiceClient, accInfoClient accountInfoClient, encodingConfig *params.EncodingConfig,
	privKey *secp256k1.PrivKey, chainID, paymentDenom string, gasPrice uint64, gasAdjustment float64) *txSender {
	return &txSender{
		txClient:       txClient,
		accInfoClient:  accInfoClient,
		encodingConfig: encodingConfig,
		privKey:        privKey,
		chainID:        chainID,
		paymentDenom:   paymentDenom,
		gasPrice:       gasPrice,
		gasAdjustment:  gasAdjustment,
	}
}

func (ts *txSender) SendTx(msgs []sdk.Msg, memo string) error {

	gasResult, err := ts.EstimateGas(msgs)
	if err != nil {
		return err
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer cancelFunc()

	accAddr, err := sdk.AccAddressFromBech32(ts.privKey.PubKey().Address().String())
	if err != nil {
		return err
	}

	accInfo, err := ts.accInfoClient.QueryInfo(ctx, accAddr.String())
	if err != nil {
		return err
	}

	signedTx, err := ts.genTx(msgs, memo, gasResult.FeeAmount, gasResult.GasLimit, accInfo.AccountNumber, accInfo.AccountSequence)
	if err != nil {
		return err
	}

	txBytes, err := ts.encodingConfig.TxConfig.TxEncoder()(signedTx)
	if err != nil {
		return err
	}

	broadcastRes, err := ts.txClient.BroadcastTx(context.Background(), &txtypes.BroadcastTxRequest{TxBytes: txBytes, Mode: txtypes.BroadcastMode_BROADCAST_MODE_BLOCK})
	if err != nil {
		return err
	}

	if broadcastRes.TxResponse == nil || broadcastRes.TxResponse.Code != 0 {
		return fmt.Errorf("broadcasting of tx failed: %+v", broadcastRes)
	}

	return nil
}

func (ts *txSender) EstimateGas(msgs []sdk.Msg) (model.GasResult, error) {

	txBuilder := ts.encodingConfig.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return model.GasResult{}, err
	}

	txBytes, err := ts.encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return model.GasResult{}, err
	}

	simRes, err := ts.txClient.Simulate(context.Background(), &txtypes.SimulateRequest{TxBytes: txBytes})
	if err != nil {
		return model.GasResult{}, err
	}

	if simRes.GasInfo == nil {
		return model.GasResult{}, errors.New("simulation result with no gas info")
	}

	estimatedGasAmount := sdk.NewIntFromUint64(uint64((float64(simRes.GasInfo.GasUsed) * ts.gasAdjustment))).Mul(sdk.NewIntFromUint64(ts.gasPrice))

	return model.GasResult{
		FeeAmount: sdk.NewCoins(sdk.NewCoin(ts.paymentDenom, estimatedGasAmount)),
		GasLimit:  uint64((float64(simRes.GasInfo.GasUsed) * ts.gasAdjustment)),
	}, nil
}

func (ts *txSender) genTx(msgs []sdk.Msg, memo string, feeAmt sdk.Coins, gas, accNum, accSeq uint64) (sdk.Tx, error) {

	signMode := ts.encodingConfig.TxConfig.SignModeHandler().DefaultMode()

	// 1st round: set SignatureV2 with empty signatures, to set correct
	// signer infos.
	sigs := []signing.SignatureV2{
		{
			PubKey: ts.privKey.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode: signMode,
			},
			Sequence: accSeq,
		},
	}

	tx := ts.encodingConfig.TxConfig.NewTxBuilder()
	if err := tx.SetMsgs(msgs...); err != nil {
		return nil, err
	}
	if err := tx.SetSignatures(sigs...); err != nil {
		return nil, err
	}

	tx.SetMemo(memo)
	tx.SetFeeAmount(feeAmt)
	tx.SetGasLimit(gas)

	// 2nd round: once all signer infos are set, every signer can sign.
	signerData := authsign.SignerData{
		ChainID:       ts.chainID,
		AccountNumber: accNum,
		Sequence:      accSeq,
	}

	signBytes, err := ts.encodingConfig.TxConfig.SignModeHandler().GetSignBytes(signMode, signerData, tx.GetTx())
	if err != nil {
		return nil, err
	}

	sig, err := ts.privKey.Sign(signBytes)
	if err != nil {
		return nil, err
	}

	sigs[0].Data.(*signing.SingleSignatureData).Signature = sig

	if err := tx.SetSignatures(sigs...); err != nil {
		return nil, err
	}

	return tx.GetTx(), nil
}

type accountInfoClient interface {
	QueryInfo(ctx context.Context, address string) (model.AccountInfo, error)
}

type txSender struct {
	txClient       txtypes.ServiceClient
	accInfoClient  accountInfoClient
	encodingConfig *params.EncodingConfig
	privKey        *secp256k1.PrivKey
	chainID        string
	paymentDenom   string
	gasPrice       uint64
	gasAdjustment  float64
}
