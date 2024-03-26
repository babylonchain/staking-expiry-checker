package types

import "github.com/btcsuite/btcd/chaincfg"

type (
	SupportedBtcNetwork string
)

const (
	BtcMainnet SupportedBtcNetwork = "mainnet"
	BtcTestnet SupportedBtcNetwork = "testnet"
	BtcSimnet  SupportedBtcNetwork = "simnet"
	BtcRegtest SupportedBtcNetwork = "regtest"
	BtcSignet  SupportedBtcNetwork = "signet"
)

func (c SupportedBtcNetwork) String() string {
	return string(c)
}

func GetBTCParams(net string) *chaincfg.Params {
	switch net {
	case BtcMainnet.String():
		return &chaincfg.MainNetParams
	case BtcTestnet.String():
		return &chaincfg.TestNet3Params
	case BtcSimnet.String():
		return &chaincfg.SimNetParams
	case BtcRegtest.String():
		return &chaincfg.RegressionNetParams
	case BtcSignet.String():
		return &chaincfg.SigNetParams
	}
	return nil
}

func GetValidNetParams() map[string]bool {
	params := map[string]bool{
		BtcMainnet.String(): true,
		BtcTestnet.String(): true,
		BtcSimnet.String():  true,
		BtcRegtest.String(): true,
	}

	return params
}
