package tx

import (
	"context"
	"errors"
	"fmt"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	client "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	auth "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"google.golang.org/grpc"
)

func NewTxSender(txClient txClient, accInfoClient accountInfoClient, encodingConfig *params.EncodingConfig,
	privKey *secp256k1.PrivKey, chainID, paymentDenom string, gasPrice uint64, gasAdjustment float64, signer signer) *txSender {
	return &txSender{
		txClient:       txClient,
		accInfoClient:  accInfoClient,
		encodingConfig: encodingConfig,
		privKey:        privKey,
		chainID:        chainID,
		paymentDenom:   paymentDenom,
		gasPrice:       gasPrice,
		gasAdjustment:  gasAdjustment,
		signer:         signer,
	}
}

func (ts *txSender) SendTx(ctx context.Context, msgs []sdk.Msg, memo string, gasResult model.GasResult) error {

	txBytes, err := ts.buildTx(ctx, msgs, memo, gasResult)
	if err != nil {
		return err
	}

	broadcastRes, err := ts.txClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{TxBytes: txBytes, Mode: txtypes.BroadcastMode_BROADCAST_MODE_BLOCK})
	if err != nil {
		return err
	}

	if broadcastRes.TxResponse == nil || broadcastRes.TxResponse.Code != 0 {
		return fmt.Errorf("broadcasting of tx failed: %+v", broadcastRes)
	}

	return nil
}

func (ts *txSender) EstimateGas(ctx context.Context, msgs []sdk.Msg, memo string) (model.GasResult, error) {

	txBytes, err := ts.buildTx(ctx, msgs, memo, model.GasResult{
		FeeAmount: sdk.NewCoins(sdk.NewCoin(ts.paymentDenom, sdk.NewInt(0))),
		GasLimit:  0,
	})
	if err != nil {
		return model.GasResult{}, err
	}

	simRes, err := ts.txClient.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: txBytes})
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

func (ts *txSender) buildTx(ctx context.Context, msgs []sdk.Msg, memo string, gasResult model.GasResult) ([]byte, error) {
	accAddr := sdk.AccAddress(ts.privKey.PubKey().Address())

	accInfo, err := ts.accInfoClient.QueryInfo(ctx, accAddr.String())
	if err != nil {
		return []byte{}, err
	}

	signedTx, err := ts.genTx(msgs, memo, gasResult.FeeAmount, gasResult.GasLimit, accInfo.AccountNumber, accInfo.AccountSequence)
	if err != nil {
		return []byte{}, err
	}

	return ts.encodingConfig.TxConfig.TxEncoder()(signedTx)
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
	if err := ts.signer.SetMsgs(tx, msgs...); err != nil {
		return nil, err
	}
	if err := ts.signer.SetSignatures(tx, sigs...); err != nil {
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

	signBytes, err := ts.signer.GetSignBytes(signMode, signerData, tx.GetTx())
	if err != nil {
		return nil, err
	}

	sig, err := ts.signer.Sign(signBytes)
	if err != nil {
		return nil, err
	}

	sigs[0].Data.(*signing.SingleSignatureData).Signature = sig

	if err := ts.signer.SetSignatures(tx, sigs...); err != nil {
		return nil, err
	}

	return tx.GetTx(), nil
}

type accountInfoClient interface {
	QueryInfo(ctx context.Context, address string) (model.AccountInfo, error)
}

type txClient interface {
	Simulate(ctx context.Context, in *txtypes.SimulateRequest, opts ...grpc.CallOption) (*txtypes.SimulateResponse, error)
	BroadcastTx(ctx context.Context, in *txtypes.BroadcastTxRequest, opts ...grpc.CallOption) (*txtypes.BroadcastTxResponse, error)
}

type signer interface {
	SetMsgs(tx client.TxBuilder, msgs ...sdk.Msg) error
	SetSignatures(tx client.TxBuilder, signatures ...signingtypes.SignatureV2) error
	GetSignBytes(mode signing.SignMode, data auth.SignerData, tx sdk.Tx) ([]byte, error)
	Sign(msg []byte) ([]byte, error)
}

type txSender struct {
	txClient       txClient
	accInfoClient  accountInfoClient
	encodingConfig *params.EncodingConfig
	privKey        *secp256k1.PrivKey
	chainID        string
	paymentDenom   string
	gasPrice       uint64
	gasAdjustment  float64
	signer         signer
}
