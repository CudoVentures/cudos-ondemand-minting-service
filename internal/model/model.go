package model

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type State struct {
	Height int64 `json:"height"`
}

type NFTData struct {
	Id              string    `json:"id"`
	Price           sdk.Int   `json:"priceInAcudos"`
	Name            string    `json:"name"`
	Uri             string    `json:"uri"`
	Data            string    `json:"data"`
	DenomID         string    `json:"denomId"`
	Status          NFTStatus `json:"status"`
	PriceValidUntil int64     `json:"priceAcudosValidUntil"`
}

func (t *NFTData) String() string {
	return fmt.Sprintf("NFTData { Id(%s) Price(%s) Name(%s) Uri(%s) Data(%s) DenomID(%s) Status(%s) PriceValidUntil(%s) }", t.Id, t.Price.String(), t.Name, t.Uri, t.Data, t.DenomID, t.Status, time.Unix(t.PriceValidUntil/1000, 0).String())
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
