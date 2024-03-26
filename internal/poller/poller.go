package poller

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-expiry-checker/internal/config"
	"github.com/babylonchain/staking-expiry-checker/internal/db"
	"github.com/babylonchain/staking-expiry-checker/internal/queue"
	"github.com/babylonchain/staking-expiry-checker/internal/queue/client"
)

type Poller struct {
	dbClient     db.DBClient
	queue        *queue.Queue
	pollInterval time.Duration
}

func NewPoller(ctx context.Context, cfg *config.Config) (*Poller, error) {
	dbClient, err := db.New(ctx, cfg.Db.DbName, cfg.Db.Address)
	if err != nil {
		return nil, err
	}

	q, err := queue.NewQueue(&cfg.Queue)

	pollInterval := cfg.Service.PollInterval
	return &Poller{
		dbClient:     dbClient,
		queue:        q,
		pollInterval: pollInterval,
	}, nil
}

func (p *Poller) Start(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.pollAndProcess(ctx)
		case <-ctx.Done():
			log.Info().Msg("Poller shutting down")
			return
		}
	}
}

func (p *Poller) pollAndProcess(ctx context.Context) {
	expiredDelegations, err := p.dbClient.FindExpiredDelegations(ctx, 0)
	if err != nil {
		log.Error().Err(err).Msg("Error finding expired delegations")
		return
	}

	for _, delegation := range expiredDelegations {
		ev := client.NewExpiredStakingEvent(delegation.StakingTxHashHex)
		err := p.queue.SendExpiredDelegationEvent(ctx, ev)
		if err != nil {
			log.Error().Err(err).Msg("Error sending expired delegation event")
			// handle the error properly, maybe retry or log
		}
	}
}
