package main

import (
	"context"
	"fmt"
	"os"

	cudosapp "github.com/CudoVentures/cudos-node/app"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	encodingconfig "github.com/CudoVentures/cudos-ondemand-minting-service/internal/encoding_config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/grpc"
	key "github.com/CudoVentures/cudos-ondemand-minting-service/internal/key"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/logger"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/marshal"
	relayminter "github.com/CudoVentures/cudos-ondemand-minting-service/internal/relay_minter"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/rpc"
	state "github.com/CudoVentures/cudos-ondemand-minting-service/internal/state"
	infraclient "github.com/CudoVentures/cudos-ondemand-minting-service/internal/tokenised_infra/client"
	"github.com/rs/zerolog"
)

func main() {
	runService(context.Background())
}

func runService(ctx context.Context) {

	cfg, err := config.NewConfig(configFilename)
	if err != nil {
		fmt.Printf("creating config failed: %s", err)
		return
	}

	cudosapp.SetConfig()
	encodingConfig := encodingconfig.MakeEncodingConfig()

	state := state.NewFileState(cfg.StateFile)

	infraClient := infraclient.NewTokenisedInfraClient(cfg.TokenisedInfraUrl, marshal.NewJsonMarshaler())

	privKey, err := key.PrivKeyFromMnemonic(cfg.WalletMnemonic)
	if err != nil {
		fmt.Println("failed to create private key from wallet mnemonic")
		return
	}

	minter := relayminter.NewRelayMinter(logger.NewLogger(zerolog.New(os.Stderr).With().Timestamp().Logger()),
		&encodingConfig, cfg, state, infraClient, privKey, grpc.GRPCConnector{}, rpc.RPCConnector{})

	minter.Start(ctx)
}

var configFilename = "config.yaml"
