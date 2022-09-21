package client

import "github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"

func NewTokenisedInfraClient() *tokenisedInfraClient {
	return &tokenisedInfraClient{}
}

func (tic *tokenisedInfraClient) GetNFTData(uid string) (model.NFTData, error) {
	return model.NFTData{}, nil
}

func (tic *tokenisedInfraClient) MarkMintedNFT(uid string) error {
	return nil
}

type tokenisedInfraClient struct {
}
