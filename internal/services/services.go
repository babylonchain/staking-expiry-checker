package services

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/babylonchain/staking-expiry-checker/internal/config"
	"github.com/babylonchain/staking-expiry-checker/internal/db"
	"github.com/babylonchain/staking-expiry-checker/internal/db/model"
	"github.com/babylonchain/staking-expiry-checker/internal/types"
)

// Service layer contains the business logic and is used to interact with
// the database and other external clients (if any).
type Services struct {
	DbClient db.DBClient
	cfg      *config.Config
}

func New(ctx context.Context, cfg *config.Config) (*Services, error) {
	dbClient, err := db.New(ctx, cfg.Db.DbName, cfg.Db.Address)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Msg("error while creating db client")
		return nil, err
	}
	return &Services{
		DbClient: dbClient,
		cfg:      cfg,
	}, nil
}

// DoHealthCheck checks the health of the services by ping the database.
func (s *Services) DoHealthCheck(ctx context.Context) error {
	return s.DbClient.Ping(ctx)
}

// ProcessExpireCheck checks if the staking delegation has expired and updates the database.
// This method tolerate duplicated calls.
func (s *Services) ProcessExpireCheck(ctx context.Context, stakingTxHashHex string, startHeight, timelock uint64) error {
	// TODO: To be implemented
	return nil
}

func (s *Services) FindExpiredDelegations(ctx context.Context, btcTipHeight uint64) (model.StakingExpiryHeightDocument, string, *types.Error) {
	resultMap, err := s.DbClient.FindExpiredDelegations(ctx, btcTipHeight)
	if err != nil {
		if db.IsInvalidPaginationTokenError(err) {
			log.Warn().Err(err).Msg("Invalid pagination token when fetching delegations by staker pk")
			return nil, "", types.NewError(http.StatusBadRequest, types.BadRequest, err)
		}
		log.Error().Err(err).Msg("Failed to find delegations by staker pk")
		return nil, "", types.NewInternalServiceError(err)
	}
	var delegations []DelegationPublic
	for _, d := range resultMap.Data {
		delegations = append(delegations, fromDelegationDocument(d))
	}
	return delegations, resultMap.PaginationToken, nil
}
