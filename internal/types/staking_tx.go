package types

type StakingTxType string

const (
	Active    StakingTxType = "active"
	Unbonding StakingTxType = "unbonding"
)

func (s StakingTxType) ToString() string {
	return string(s)
}
