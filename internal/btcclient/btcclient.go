package btcclient

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"

	"github.com/babylonchain/staking-expiry-checker/internal/config"
	"github.com/babylonchain/staking-expiry-checker/internal/observability/metrics"
	"github.com/babylonchain/staking-expiry-checker/internal/utils"
)

type BtcClient struct {
	client *rpcclient.Client

	params *chaincfg.Params
	cfg    *config.BtcConfig
}

func NewBtcClient(cfg *config.BtcConfig) (*BtcClient, error) {
	params := utils.GetBTCParams(cfg.NetParams)

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
		client: rpcClient,
		params: params,
		cfg:    cfg,
	}, nil
}

func (b *BtcClient) GetBlockCount() (int64, error) {
	return metrics.RecordBtcClientMetrics[int64](b.client.GetBlockCount)
}
