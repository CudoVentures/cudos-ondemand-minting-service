package client

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/rs/zerolog/log"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
)

func NewTokenisedInfraClient(url string, marshaler marshaler) *tokenisedInfraClient {
	return &tokenisedInfraClient{
		url:       url,
		client:    &http.Client{Timeout: clientTimeout},
		marshaler: marshaler,
	}
}

func (tic *tokenisedInfraClient) GetNFTData(ctx context.Context, cfg config.Config, uid, recipientCudosAddress string, paidAmount sdk.Coin) (model.NFTData, error) {
	log.Info().Msgf("making request to %s", fmt.Sprintf("%s%s/%s/%s/%s", tic.url, getNFTDataUri, uid, recipientCudosAddress, paidAmount.Amount))
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s%s/%s/%s/%s", tic.url, getNFTDataUri, uid, recipientCudosAddress, paidAmount.Amount),
		nil,
	)

	if err != nil {
		return model.NFTData{}, err
	}

	req.Header.Set("aura-pool-api-key", cfg.AuraPoolApiKey)

	res, err := tic.client.Do(req)
	if err != nil {
		return model.NFTData{}, err
	}

	if res.StatusCode != http.StatusOK {
		return model.NFTData{}, nil
	}

	return tic.parseBody(res)
}

func (tic *tokenisedInfraClient) parseBody(res *http.Response) (model.NFTData, error) {
	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	// log.Info().Msgf("received NFT Data %s", string(body))
	if err != nil {
		return model.NFTData{}, err
	}
	nft := model.NFTData{}
	if err := tic.marshaler.Unmarshal(body, &nft); err != nil {
		return model.NFTData{}, err
	}

	return nft, nil
}

type marshaler interface {
	Unmarshal(data []byte, v any) error
	Marshal(v any) ([]byte, error)
}

type mintTx struct {
	TxHash string `json:"tx_hash"`
	Uid    string `json:"uid"`
}

type tokenisedInfraClient struct {
	url       string
	client    *http.Client
	marshaler marshaler
}

const (
	clientTimeout = time.Second * 10
	getNFTDataUri = "/api/v1/nft/on-demand-minting-nft"
)
