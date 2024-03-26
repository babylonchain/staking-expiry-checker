package model

const StakingExpiryHeightsCollection = "staking_expiry_heights"

type StakingExpiryHeightDocument struct {
	StakingTxHashHex string `bson:"_id"` // Primary key
	ExpireBtcHeight  uint64 `bson:"expire_btc_height"`
}
