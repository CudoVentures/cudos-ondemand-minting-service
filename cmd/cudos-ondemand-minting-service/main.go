package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	cudosapp "github.com/CudoVentures/cudos-node/app"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	encodingconfig "github.com/CudoVentures/cudos-ondemand-minting-service/internal/encoding_config"
	key "github.com/CudoVentures/cudos-ondemand-minting-service/internal/key"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/logger"
	relayminter "github.com/CudoVentures/cudos-ondemand-minting-service/internal/relay_minter"
	state "github.com/CudoVentures/cudos-ondemand-minting-service/internal/state"
	infraclient "github.com/CudoVentures/cudos-ondemand-minting-service/internal/tokenised_infra/client"
	"github.com/rs/zerolog"
)

func main() {
	runService(context.Background())
}

func runService(ctx context.Context) {
	cfg, err := config.NewConfig("config.yaml")

	zlogger := logger.NewLogger(zerolog.New(os.Stderr).With().Timestamp().Logger())

	if err != nil {
		zlogger.Fatal(fmt.Errorf("creating config failed: %s", err))
		return
	}

	cudosapp.SetConfig()
	encodingConfig := encodingconfig.MakeEncodingConfig()

	state := state.NewFileState(cfg.StateFile)

	infraClient := infraclient.NewTokenisedInfraClient(cfg.TokenisedInfraUrl)

	privKey, err := key.PrivKeyFromMnemonic(cfg.WalletMnemonic)
	if err != nil {
		zlogger.Fatal(errors.New("failed to create private key from wallet mnemonic"))
		return
	}

	minter := relayminter.NewRelayMinter(zlogger, &encodingConfig, cfg, state, infraClient, privKey)

	minter.Start(ctx)
}
