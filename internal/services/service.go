package services

import (
	"context"

	"github.com/babylonchain/staking-expiry-checker/internal/btcclient"
	"github.com/babylonchain/staking-expiry-checker/internal/db"
	"github.com/babylonchain/staking-expiry-checker/internal/queue"
	queueclient "github.com/babylonchain/staking-expiry-checker/internal/queue/client"
)

type Service struct {
	db           db.DbInterface
	btc          btcclient.BtcInterface
	queueManager *queue.QueueManager
}

func NewService(db db.DbInterface, btc btcclient.BtcInterface, qm *queue.QueueManager) *Service {
	return &Service{
		db:           db,
		btc:          btc,
		queueManager: qm,
	}
}

func (s *Service) ProcessExpiredDelegations(ctx context.Context) error {
	btcTip, err := s.btc.GetBlockCount()
	if err != nil {
		return err
	}

	expiredDelegations, err := s.db.FindExpiredDelegations(ctx, uint64(btcTip))
	if err != nil {
		return err
	}

	for _, delegation := range expiredDelegations {
		ev := queueclient.NewExpiredStakingEvent(delegation.StakingTxHashHex)
		if err := s.queueManager.SendExpiredDelegationEvent(ctx, ev); err != nil {
			return err
		}
	}

	return nil
}
