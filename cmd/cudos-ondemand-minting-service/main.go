package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

	cudosapp "github.com/CudoVentures/cudos-node/app"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	encodingconfig "github.com/CudoVentures/cudos-ondemand-minting-service/internal/encoding_config"
	key "github.com/CudoVentures/cudos-ondemand-minting-service/internal/key"
	relayminter "github.com/CudoVentures/cudos-ondemand-minting-service/internal/relay_minter"
	filestate "github.com/CudoVentures/cudos-ondemand-minting-service/internal/state/file"
	infraclient "github.com/CudoVentures/cudos-ondemand-minting-service/internal/tokenised_infra/client"
)

func main() {
	cfg, err := config.NewConfig("config.yaml")
	if err != nil {
		log.Fatal().Err(fmt.Errorf("creating config failed: %s", err)).Send()
		return
	}

	cudosapp.SetConfig()
	encodingConfig := encodingconfig.MakeEncodingConfig()

	state := filestate.NewFileState(cfg.StateFile)

	infraClient := infraclient.NewTokenisedInfraClient()

	privKey, err := key.PrivKeyFromMnemonic(cfg.WalletMnemonic)
	if err != nil {
		log.Fatal().Err(errors.New("failed to create private key from wallet mnemonic"))
		return
	}

	minter := relayminter.NewRelayMinter(encodingConfig, cfg, state, infraClient, privKey)

	err = minter.Start(context.Background())

	log.Fatal().Err(err).Send()
}
