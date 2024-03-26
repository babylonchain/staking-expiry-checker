package main

import (
	"context"
	"fmt"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-expiry-checker/cmd/staking-expiry-checker/cli"
	"github.com/babylonchain/staking-expiry-checker/internal/api"
	"github.com/babylonchain/staking-expiry-checker/internal/config"
	"github.com/babylonchain/staking-expiry-checker/internal/observability/metrics"
	"github.com/babylonchain/staking-expiry-checker/internal/queue"
	"github.com/babylonchain/staking-expiry-checker/internal/services"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Debug().Msg("failed to load .env file")
	}
}

func main() {
	ctx := context.Background()

	// setup cli commands and flags
	if err := cli.Setup(); err != nil {
		log.Fatal().Err(err).Msg("error while setting up cli")
	}

	// load config
	cfgPath := cli.GetConfigPath()
	cfg, err := config.New(cfgPath)
	if err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("error while loading config file: %s", cfgPath))
	}

	// initialize metrics with the metrics port from config
	metricsPort := cfg.Metrics.GetMetricsPort()
	metrics.Init(metricsPort)

	services, err := services.New(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("error while setting up staking services layer")
	}
	// Start the event queue processing
	queues := queue.New(cfg.Queue, services)
	queues.StartReceivingMessages()

	apiServer, err := api.New(ctx, cfg, services)
	if err != nil {
		log.Fatal().Err(err).Msg("error while setting up staking api service")
	}
	if err = apiServer.Start(); err != nil {
		log.Fatal().Err(err).Msg("error while starting staking api service")
	}
}
