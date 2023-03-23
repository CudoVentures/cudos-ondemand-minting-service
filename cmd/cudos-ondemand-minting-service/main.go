package main

import (
	"context"
	"os"

	cudosapp "github.com/CudoVentures/cudos-node/app"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/email"
	encodingconfig "github.com/CudoVentures/cudos-ondemand-minting-service/internal/encoding_config"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/grpc"
	key "github.com/CudoVentures/cudos-ondemand-minting-service/internal/key"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/logger"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/marshal"
	relayminter "github.com/CudoVentures/cudos-ondemand-minting-service/internal/relay_minter"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/rpc"
	state "github.com/CudoVentures/cudos-ondemand-minting-service/internal/state"
	infraclient "github.com/CudoVentures/cudos-ondemand-minting-service/internal/tokenised_infra/client"
	"github.com/CudoVentures/cudos-ondemand-minting-service/internal/tx"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// The one and only entrypoint of the program.
func main() {
	runService(context.Background())
}

// This function does initial params processing and stars the relayer thread at the end.
// The initial process included following:
//
// - Parsing config file;
//
// - Creating logger instnaces;
//
// - Loading state file.
// The state file is a JSON object with 'height' property and nothing else. It indicates the starting block of the service.
// Without this state the service should starts always from block 1 therefore processing everything from 1 to current network height each time.
// Although it is completely safe to process a block multiple times it is just a waste of time. So state is used in order not to waste time for processing already processed blocks.
//
// - Creating AuraPool client;
//
// - Creating Relayer instance.
//
// At the end of the function the so called Relayer starts in a thread
func runService(ctx context.Context) {
	cfg, err := config.NewConfig(envPath)
	if err != nil {
		log.Fatal().Msgf("creating config failed: %s", err)
		return
	}

	var rmLogger zerolog.Logger
	if cfg.HasPrettyLogging() {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		rmLogger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		rmLogger = zerolog.New(os.Stderr)
	}

	log.Info().Msgf("starting on-demand-minting-service using config %s", cfg.String())

	cudosapp.SetConfig()
	encodingConfig := encodingconfig.MakeEncodingConfig()

	state := state.NewFileState(cfg.StateFile)

	infraClient := infraclient.NewTokenisedInfraClient(cfg.AuraPoolBackend, marshal.NewJsonMarshaler())

	privKey, err := key.PrivKeyFromMnemonic(cfg.WalletMnemonic)
	if err != nil {
		log.Error().Msg("failed to create private key from wallet mnemonic")
		return
	}

	rm := relayminter.NewRelayMinter(
		logger.NewLogger(rmLogger.With().Str("module", "relayer").Timestamp().Logger()),
		&encodingConfig,
		cfg,
		state,
		infraClient,
		privKey,
		grpc.GRPCConnector{},
		rpc.RPCConnector{},
		tx.NewTxCoder(&encodingConfig),
		email.NewSendgridEmailService(cfg),
	)

	go rm.Start(ctx)

	// log.Info().Msg("registering http handlers")

	// r := mux.NewRouter()
	// r.HandleFunc("/simulate/mint", handlers.GetMintTxFee(cfg, sdk.AccAddress(privKey.PubKey().Address()).String(), rm, rm))

	// log.Info().Msg(fmt.Sprintf("listening on port %d", cfg.Port))
	// srv := &http.Server{
	// 	Handler:      r,
	// 	Addr:         fmt.Sprintf(":%d", cfg.Port),
	// 	WriteTimeout: 15 * time.Second,
	// 	ReadTimeout:  15 * time.Second,
	// }

	// go func() {
	// 	if err := srv.ListenAndServe(); err != nil {
	// 		log.Fatal().Msgf("error while listening: %s", err)
	// 	}
	// }()

	<-ctx.Done()

	// srv.Shutdown(context.Background())
	log.Info().Msg("stopping on-demand-minting-service")
}

var envPath = ".env"
