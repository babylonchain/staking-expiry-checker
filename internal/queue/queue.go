package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-expiry-checker/internal/config"
	"github.com/babylonchain/staking-expiry-checker/internal/queue/client"
)

type Queue struct {
	stakingExpiredEventQueue client.QueueClient
}

func NewQueue(cfg *config.QueueConfig) (*Queue, error) {
	stakingEventQueue, err := client.NewQueueClient(cfg.Url, cfg.QueueUser, cfg.QueuePassword, client.ExpiredStakingQueueName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize staking event queue: %w", err)
	}

	return &Queue{
		stakingExpiredEventQueue: stakingEventQueue,
	}, nil
}

func (q *Queue) SendExpiredDelegationEvent(ctx context.Context, ev client.ExpiredStakingEvent) error {
	jsonBytes, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	messageBody := string(jsonBytes)

	log.Debug().Str("tx_hash", ev.StakingTxHashHex).Msg("publishing expired staking event")
	err = q.stakingExpiredEventQueue.SendMessage(ctx, messageBody)
	if err != nil {
		return fmt.Errorf("failed to publish staking event: %w", err)
	}
	log.Debug().Str("tx_hash", ev.StakingTxHashHex).Msg("successfully published expired staking event")

	return nil
}

// Shutdown gracefully stops the interaction with the queue, ensuring all resources are properly released.
func (q *Queue) Shutdown() error {
	q.stakingExpiredEventQueue.Stop()
	return nil
}
