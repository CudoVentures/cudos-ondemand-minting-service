package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	cudosapp "github.com/CudoVentures/cudos-node/app"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	encodingconfig "github.com/CudoVentures/cudos-ondemand-minting-service/internal/encoding_config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/grpc"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/handlers"
	key "github.com/CudoVentures/cudos-ondemand-minting-service/internal/key"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/logger"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/marshal"
	relayminter "github.com/CudoVentures/cudos-ondemand-minting-service/internal/relay_minter"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/rpc"
	state "github.com/CudoVentures/cudos-ondemand-minting-service/internal/state"
	infraclient "github.com/CudoVentures/cudos-ondemand-minting-service/internal/tokenised_infra/client"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	runService(context.Background())
}

func runService(ctx context.Context) {

	cfg, err := config.NewConfig(envPath)
	if err != nil {
		fmt.Printf("creating config failed: %s", err)
		return
	}

	cudosapp.SetConfig()
	encodingConfig := encodingconfig.MakeEncodingConfig()

	state := state.NewFileState(cfg.StateFile)

	infraClient := infraclient.NewTokenisedInfraClient(cfg.AuraPoolBackend, marshal.NewJsonMarshaler())

	privKey, err := key.PrivKeyFromMnemonic(cfg.WalletMnemonic)
	if err != nil {
		fmt.Println("failed to create private key from wallet mnemonic")
		return
	}

	rm := relayminter.NewRelayMinter(logger.NewLogger(zerolog.New(os.Stderr).With().Timestamp().Logger()),
		&encodingConfig, cfg, state, infraClient, privKey, grpc.GRPCConnector{}, rpc.RPCConnector{}, tx.NewTxCoder(&encodingConfig))

	go rm.Start(ctx)

	log.Info().Msg("Registering http handlers")

	r := mux.NewRouter()
	r.HandleFunc("/simulate/mint", handlers.GetMintTxFee(cfg, sdk.AccAddress(privKey.PubKey().Address()).String(), rm, rm))

	log.Info().Msg(fmt.Sprintf("Listening on port: %d", cfg.Port))
	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal().Err(fmt.Errorf("error while listening: %s", err))
		}
	}()

	<-ctx.Done()

	srv.Shutdown(context.Background())
}

var envPath = ".env"
