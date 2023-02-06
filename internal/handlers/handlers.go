package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	marketplacetypes "github.com/CudoVentures/cudos-node/x/marketplace/types"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GetMintTxFee(cfg config.Config, walletAddress string, ge gasEstmator, ndp nftDataProvider) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		uid := r.URL.Query().Get("uid")
		if uid == "" {
			badRequest(w, fmt.Errorf("empty uid"))
			return
		}

		recipient := r.URL.Query().Get("recipient")
		if recipient == "" {
			badRequest(w, fmt.Errorf("empty recipient"))
			return
		}

		nftData, err := ndp.GetNFTData(r.Context(), uid, recipient)
		if err != nil {
			badRequest(w, err)
			return
		}

		nftData.Price = sdk.NewIntFromUint64(1000000000000000000)

		msgMintNft := marketplacetypes.NewMsgMintNft(walletAddress, nftData.DenomID, recipient, nftData.Name, nftData.Uri, nftData.Data, uid, sdk.NewCoin("acudos", nftData.Price))
		gasResult, err := ge.EstimateGas(r.Context(), []sdk.Msg{msgMintNft}, "")
		if err != nil {
			badRequest(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(gasResultResponse{
			FeeAmount: gasResult.FeeAmount,
			GasLimit:  gasResult.GasLimit,
		}); err != nil {
			fmt.Println(err)
		}
	}
}

func badRequest(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	if err := json.NewEncoder(w).Encode(badRequestResponse{
		Error: err.Error(),
	}); err != nil {
		fmt.Println(err)
	}
}

type badRequestResponse struct {
	Error string `json:"error"`
}

type gasEstmator interface {
	EstimateGas(ctx context.Context, msgs []sdk.Msg, memo string) (model.GasResult, error)
}

type nftDataProvider interface {
	GetNFTData(ctx context.Context, uid, recipientCudosAddress string) (model.NFTData, error)
}

type gasResultResponse struct {
	FeeAmount sdk.Coins `json:"feeAmount"`
	GasLimit  uint64    `json:"gasLimit"`
}
