package model

import sdk "github.com/cosmos/cosmos-sdk/types"

type State struct {
	Height int64 `json:"height"`
}

type NFTData struct {
	Price   sdk.Coin  `json:"price"`
	Name    string    `json:"name"`
	Uri     string    `json:"uri"`
	Data    string    `json:"data"`
	DenomID string    `json:"denom_id"`
	Status  NFTStatus `json:"state"`
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
