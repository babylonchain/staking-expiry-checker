package main

import (
	"context"
	"fmt"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-expiry-checker/cmd/staking-expiry-checker/cli"
	"github.com/babylonchain/staking-expiry-checker/internal/btcclient"
	"github.com/babylonchain/staking-expiry-checker/internal/config"
	"github.com/babylonchain/staking-expiry-checker/internal/db"
	"github.com/babylonchain/staking-expiry-checker/internal/observability/metrics"
	"github.com/babylonchain/staking-expiry-checker/internal/poller"
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

	// create new db client
	dbClient, err := db.New(ctx, cfg.Db.DbName, cfg.Db.Address)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating db client")
	}

	btcClient, err := btcclient.NewBtcClient(&cfg.Btc)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating btc client")
	}

	qm, err := queue.NewQueueManager(&cfg.Queue)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating queue manager")
	}

	delegationService := services.NewService(dbClient, btcClient, qm)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating delegation service")
	}

	p, err := poller.NewPoller(cfg.Poller.Interval, delegationService)
	if err != nil {
		log.Fatal().Err(err).Msg("error while creating poller")
	}
	p.Start(ctx)
}
