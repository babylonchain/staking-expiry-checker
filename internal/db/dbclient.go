package db

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	opts := options.Find().SetLimit(10)
	cursor, err := client.Find(ctx, filter, opts)
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

func (db *Database) DeleteExpiredDelegation(ctx context.Context, id primitive.ObjectID) error {
	client := db.client.Database(db.dbName).Collection(model.TimeLockCollection)
	filter := bson.M{"_id": id}

	result, err := client.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete expired delegation with ID %v: %w", id, err)
	}

	// Check if any document was deleted
	if result.DeletedCount == 0 {
		return fmt.Errorf("no expired delegation found with ID %v", id)
	}

	return nil
}
