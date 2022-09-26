package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/rs/zerolog/log"

	cudosapp "github.com/CudoVentures/cudos-node/app"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
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
	encodingConfig := makeEncodingConfig([]module.BasicManager{cudosapp.ModuleBasics})()

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

// func initializeEncodingConfig() (params.EncodingConfig, error) {
// dir, err := ioutil.TempDir("", "ondemand-minting")
// if err != nil {
// 	return params.EncodingConfig{}, err
// }

// db, err := sdk.NewLevelDB("ondemand-minting", dir)
// if err != nil {
// 	return params.EncodingConfig{}, err
// }

// userHomeDir, err := os.Getwd()
// if err != nil {
// 	return params.EncodingConfig{}, err
// }

// defaultNodeHome := filepath.Join(userHomeDir, "cudos-data")

// TODO: Replace with proper logger
// _ = simapp.NewSimApp(tendermintlogger.NewNopLogger(), db, nil, true, map[int64]bool{}, defaultNodeHome, sdksimapp.FlagPeriodValue, encodingConfig, simapp.EmptyAppOptions{})

// 	return encodingConfig, nil
// }

func makeEncodingConfig(managers []module.BasicManager) func() params.EncodingConfig {
	return func() params.EncodingConfig {
		encodingConfig := params.MakeTestEncodingConfig()
		std.RegisterLegacyAminoCodec(encodingConfig.Amino)
		std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
		manager := mergeBasicManagers(managers)
		manager.RegisterLegacyAminoCodec(encodingConfig.Amino)
		manager.RegisterInterfaces(encodingConfig.InterfaceRegistry)
		return encodingConfig
	}
}

// mergeBasicManagers merges the given managers into a single module.BasicManager
func mergeBasicManagers(managers []module.BasicManager) module.BasicManager {
	var union = module.BasicManager{}
	for _, manager := range managers {
		for k, v := range manager {
			union[k] = v
		}
	}
	return union
}
