package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/model"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/nft/minted/check-status", getNFTMintedHandler())
	r.HandleFunc("/api/v1/nft/on-demand-minting-nft/{uid}/{recipient}/{amount}", getNFTHandler())

	log.Info().Msg(fmt.Sprintf("Listening on port: %d", listeningPort))
	srv := &http.Server{
		Handler: r,
		Addr:    fmt.Sprintf(":%d", listeningPort),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal().Err(fmt.Errorf("error while listening: %s", err)).Send()
	}
}

func getNFTMintedHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error().Err(fmt.Errorf("error while reading body: %s", err)).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var nftUidData mintTx
		if err := json.Unmarshal(body, &nftUidData); err != nil {
			log.Error().Err(fmt.Errorf("error while unmarshalling body: %s", err)).Send()
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if _, ok := nfts[nftUidData.Uid]; !ok {
			log.Error().Err(fmt.Errorf("nft with uid (%s) not found", nftUidData.Uid)).Send()
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
	}
}

func getNFTHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := mux.Vars(r)["uid"]
		nft, ok := nfts[uid]
		if !ok {
			log.Error().Err(fmt.Errorf("nft with uid (%s) not found", uid)).Send()
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(nft); err != nil {
			log.Error().Err(err).Send()
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

var nfts = map[string]model.NFTData{
	"nftuid1": {
		Price:           sdk.NewIntFromUint64(8000000000000000000),
		Name:            "test nft name",
		Uri:             "test nft uri",
		Data:            "test nft data",
		DenomID:         "testdenom",
		Status:          model.QueuedNFTStatus,
		PriceValidUntil: tomorrow,
	},
	"nftuid2": {
		Price:           sdk.NewIntFromUint64(8000000000000000000),
		Name:            "test nft name",
		Uri:             "test nft uri",
		Data:            "test nft data",
		DenomID:         "testdenom",
		Status:          model.RejectedNFTStatus,
		PriceValidUntil: tomorrow,
	},
}

const listeningPort = 8080

var tomorrow = time.Now().Add(time.Hour * 24).UnixMilli()

type mintTx struct {
	TxHash string `json:"tx_hash"`
	Uid    string `json:"uuid"`
}
