package client

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
)

func NewTokenisedInfraClient(url string, marshaler marshaler) *tokenisedInfraClient {
	return &tokenisedInfraClient{
		url:       url,
		client:    &http.Client{Timeout: clientTimeout},
		marshaler: marshaler,
	}
}

func (tic *tokenisedInfraClient) GetNFTData(ctx context.Context, uid string) (model.NFTData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s%s?uid=%s", tic.url, getNFTDataUri, uid), nil)
	if err != nil {
		return model.NFTData{}, err
	}

	res, err := tic.client.Do(req)
	if err != nil {
		return model.NFTData{}, err
	}

	if res.StatusCode == http.StatusNotFound {
		return model.NFTData{}, nil
	}

	return tic.parseBody(res)
}

func (tic *tokenisedInfraClient) MarkMintedNFT(ctx context.Context, mintTxHash, uid string) error {
	markNftData, err := tic.marshaler.Marshal(mintTx{
		TxHash: mintTxHash,
		Uid:    uid,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("%s%s", tic.url, markMintedNFTUri), bytes.NewBuffer(markNftData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	res, err := tic.client.Do(req)
	if err != nil {
		return err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	return nil
}

func (tic *tokenisedInfraClient) parseBody(res *http.Response) (model.NFTData, error) {
	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
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
	clientTimeout    = time.Second * 10
	getNFTDataUri    = "/nft"
	markMintedNFTUri = "/nft/minted/check-status"
)
