package db

import (
	"context"

	"github.com/babylonchain/staking-expiry-checker/internal/db/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DbInterface interface {
	Ping(ctx context.Context) error
	FindExpiredDelegations(
		ctx context.Context, btcTipHeight uint64,
	) ([]model.TimeLockDocument, error)
	DeleteExpiredDelegation(
		ctx context.Context, id primitive.ObjectID,
	) error
}
