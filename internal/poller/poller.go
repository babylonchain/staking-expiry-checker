package poller

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-expiry-checker/internal/btcclient"
	"github.com/babylonchain/staking-expiry-checker/internal/config"
	"github.com/babylonchain/staking-expiry-checker/internal/db"
	"github.com/babylonchain/staking-expiry-checker/internal/queue"
	"github.com/babylonchain/staking-expiry-checker/internal/queue/client"
)

type Poller struct {
	dbClient  db.DBClient
	btcClient *btcclient.BtcClient
	queue     *queue.Queue
	cfg       *config.PollerConfig

	quit chan struct{}
}

func NewPoller(ctx context.Context, cfg *config.Config) (*Poller, error) {
	dbClient, err := db.New(ctx, cfg.Db.DbName, cfg.Db.Address)
	if err != nil {
		return nil, err
	}

	bc, err := btcclient.New(&cfg.Btc)
	if err != nil {
		return nil, err
	}

	q, err := queue.NewQueue(&cfg.Queue)
	return &Poller{
		dbClient:  dbClient,
		btcClient: bc,
		queue:     q,
		cfg:       &cfg.Poller,
	}, nil
}

func (p *Poller) Start(ctx context.Context) {
	ticker := time.NewTicker(p.cfg.PollInterval)

	for {
		select {
		case <-ticker.C:
			err := p.pollAndProcess(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Error polling and processing")
			}

		case <-p.quit:
			ticker.Stop() // Stop the ticker
			return
		}
	}
}

func (p *Poller) Stop() {
	close(p.quit)
}

func (p *Poller) pollAndProcess(ctx context.Context) error {
	btcTip, err := p.btcClient.GetBlockCount()
	if err != nil {
		log.Error().Err(err).Msg("Error getting btc tip")
		return err
	}

	expiredDelegations, err := p.dbClient.FindExpiredDelegations(ctx, uint64(btcTip))
	if err != nil {
		log.Error().Err(err).Msg("Error finding expired delegations")
		return err
	}

	for _, delegation := range expiredDelegations {
		ev := client.NewExpiredStakingEvent(delegation.StakingTxHashHex)
		err := p.queue.SendExpiredDelegationEvent(ctx, ev)
		if err != nil {
			log.Error().Err(err).Msg("Error sending expired delegation event to queue")
			return err
		}
	}

	return nil
}
