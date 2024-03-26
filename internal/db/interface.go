package db

import (
	"context"

	"github.com/babylonchain/staking-expiry-checker/internal/db/model"
)

type DBClient interface {
	Ping(ctx context.Context) error
	FindExpiredDelegations(
		ctx context.Context, btcTipHeight uint64,
	) (*model.StakingExpiryHeightDocument, error)
}
