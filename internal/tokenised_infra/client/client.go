package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
)

func NewTokenisedInfraClient(url string) *tokenisedInfraClient {
	return &tokenisedInfraClient{
		url:    url,
		client: &http.Client{Timeout: clientTimeout},
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

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return model.NFTData{}, err
	}

	nft := model.NFTData{}
	if err := json.Unmarshal(body, &nft); err != nil {
		return model.NFTData{}, err
	}

	return nft, nil
}

// TODO: This call should be authenticated

func (tic *tokenisedInfraClient) MarkMintedNFT(ctx context.Context, uid string) error {
	markNftData, err := json.Marshal(nftUid{UID: uid})
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

type nftUid struct {
	UID string `json:"uid"`
}

type tokenisedInfraClient struct {
	url    string
	client *http.Client
}

const (
	clientTimeout    = time.Second * 10
	getNFTDataUri    = "/nft"
	markMintedNFTUri = "/nft/minted"
)
