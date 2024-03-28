package db

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/babylonchain/staking-expiry-checker/internal/db/model"
)

type Database struct {
	dbName string
	client *mongo.Client
}

func New(ctx context.Context, dbName string, dbURI string) (*Database, error) {
	clientOps := options.Client().ApplyURI(dbURI)
	client, err := mongo.Connect(ctx, clientOps)
	if err != nil {
		return nil, err
	}

	return &Database{
		dbName: dbName,
		client: client,
	}, nil
}

func (db *Database) Ping(ctx context.Context) error {
	err := db.client.Ping(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

func (db *Database) FindExpiredDelegations(ctx context.Context, btcTipHeight uint64) ([]model.TimeLockDocument, error) {
	client := db.client.Database(db.dbName).Collection(model.TimeLockCollection)
	filter := bson.M{"expire_height": bson.M{"$lte": btcTipHeight}}

	cursor, err := client.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var delegations []model.TimeLockDocument
	if err = cursor.All(ctx, &delegations); err != nil {
		return nil, err
	}

	return delegations, nil
}

// DeleteExpiredDelegation deletes a delegation identified by its staking transaction hash.
func (db *Database) DeleteExpiredDelegation(ctx context.Context, stakingTxHashHex string) error {
	client := db.client.Database(db.dbName).Collection(model.TimeLockCollection)
	filter := bson.M{"staking_tx_hash_hex": stakingTxHashHex}

	result, err := client.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete expired delegation %s: %w", stakingTxHashHex, err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("no expired delegation found with tx hash %s", stakingTxHashHex)
	}

	return nil
}
