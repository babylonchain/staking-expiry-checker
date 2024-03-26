package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/babylonchain/staking-expiry-checker/internal/db/model"
)

type Database struct {
	DbName string
	Client *mongo.Client
}

type DbResultMap[T any] struct {
	Data            []T    `json:"data"`
	PaginationToken string `json:"paginationToken"`
}

func New(ctx context.Context, dbName string, dbURI string) (*Database, error) {
	clientOps := options.Client().ApplyURI(dbURI)
	client, err := mongo.Connect(ctx, clientOps)
	if err != nil {
		return nil, err
	}

	return &Database{
		DbName: dbName,
		Client: client,
	}, nil
}

func (db *Database) Ping(ctx context.Context) error {
	err := db.Client.Ping(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

func (db *Database) FindExpiredDelegations(ctx context.Context, btcTipHeight uint64) ([]model.StakingExpiryHeightDocument, error) {
	client := db.Client.Database(db.DbName).Collection(model.StakingExpiryHeightsCollection)

	filter := bson.M{"expire_btc_height": bson.M{"$lte": btcTipHeight}}

	cursor, err := client.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var delegations []model.StakingExpiryHeightDocument
	if err = cursor.All(ctx, &delegations); err != nil {
		return nil, err
	}

	return delegations, nil
}
