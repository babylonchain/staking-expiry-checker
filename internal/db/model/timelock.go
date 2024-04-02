package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const TimeLockCollection = "timelock_queue"

type TimeLockDocument struct {
	ID               primitive.ObjectID `bson:"_id"`
	StakingTxHashHex string             `bson:"staking_tx_hash_hex"`
	ExpireHeight     uint64             `bson:"expire_height"`
	TxType           string             `bson:"tx_type"`
}
