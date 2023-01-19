package model

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type State struct {
	Height int64 `json:"height"`
}

type NFTData struct {
	Price   sdk.Int   `json:"priceInAcudos"`
	Name    string    `json:"name"`
	Uri     string    `json:"uri"`
	Data    string    `json:"data"`
	DenomID string    `json:"denomId"`
	Status  NFTStatus `json:"status"`
}

type NFTStatus string

const (
	QueuedNFTStatus   NFTStatus = "queued"
	ApprovedNFTStatus           = "approved"
	RejectedNFTStatus           = "rejected"
	ExpiredNFTStatus            = "expired"
	DeletedNFTStatus            = "deleted"
)

type AccountInfo struct {
	AccountNumber   uint64
	AccountSequence uint64
}

type GasResult struct {
	FeeAmount sdk.Coins
	GasLimit  uint64
}
