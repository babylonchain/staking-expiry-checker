package btcclient

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"

	"github.com/babylonchain/staking-expiry-checker/internal/config"
	"github.com/babylonchain/staking-expiry-checker/internal/types"
)

type BtcClient struct {
	Client *rpcclient.Client

	Params *chaincfg.Params
	Cfg    *config.BtcConfig
}

func New(cfg *config.BtcConfig) (*BtcClient, error) {
	params := types.GetBTCParams(cfg.NetParams)

	connCfg := &rpcclient.ConnConfig{
		Host:         cfg.Endpoint,
		HTTPPostMode: true,
		User:         cfg.RpcUser,
		Pass:         cfg.RpcPass,
		DisableTLS:   cfg.DisableTLS,
		Params:       params.Name,
	}

	rpcClient, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, err
	}

	return &BtcClient{
		Client: rpcClient,
		Params: params,
		Cfg:    cfg,
	}, nil
}
