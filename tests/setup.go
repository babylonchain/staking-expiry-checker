package tests

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/babylonchain/staking-expiry-checker/internal/btcclient"
	"github.com/babylonchain/staking-expiry-checker/internal/config"
	"github.com/babylonchain/staking-expiry-checker/internal/db"
	"github.com/babylonchain/staking-expiry-checker/internal/db/model"
	"github.com/babylonchain/staking-expiry-checker/internal/observability/metrics"
	"github.com/babylonchain/staking-expiry-checker/internal/poller"
	"github.com/babylonchain/staking-expiry-checker/internal/queue"
	"github.com/babylonchain/staking-expiry-checker/internal/services"
	"github.com/babylonchain/staking-queue-client/client"

	queueconfig "github.com/babylonchain/staking-queue-client/config"
)

type TestServerDependency struct {
	ConfigOverrides *config.Config
	MockDbClient    db.DbInterface
	MockBtcClient   btcclient.BtcInterface
}

func setupTestServer(t *testing.T, dep *TestServerDependency) (*queue.QueueManager, *amqp091.Connection, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	cfg, err := config.New("./config-test.yml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	metricsPort := cfg.Metrics.GetMetricsPort()
	metrics.Init(metricsPort)

	if dep != nil && dep.ConfigOverrides != nil {
		applyConfigOverrides(cfg, dep.ConfigOverrides)
	}

	qm, conn, err := setUpTestQueue(t, &cfg.Queue)
	if err != nil {
		t.Fatalf("Failed to setup test queue: %v", err)
	}

	var (
		dbClient  db.DbInterface
		btcClient btcclient.BtcInterface
	)

	if dep != nil && dep.MockBtcClient != nil {
		btcClient = dep.MockBtcClient
	} else {
		btcClient, err = btcclient.NewBtcClient(&cfg.Btc)
		if err != nil {
			t.Fatalf("Failed to initialize btc client: %v", err)
		}
	}

	if dep != nil && dep.MockDbClient != nil {
		dbClient = dep.MockDbClient
	} else {
		setupTestDB(cfg)
		dbClient, err = db.New(ctx, cfg.Db.DbName, cfg.Db.Address)
		if err != nil {
			t.Fatalf("Failed to initialize db client: %v", err)
		}

	}

	service := services.NewService(dbClient, btcClient, qm)
	p, err := poller.NewPoller(cfg.Poller.Interval, service)
	if err != nil {
		t.Fatalf("Failed to initialize poller: %v", err)
	}

	teardown := func() {
		p.Stop()
		qm.Shutdown()
		err := conn.Close()
		if err != nil {
			log.Fatal("Failed to close connection to RabbitMQ: ", err)
		}
		cancel() // Cancel the context to release resources
	}

	go p.Start(ctx)
	return qm, conn, teardown
}

// Generic function to apply configuration overrides
func applyConfigOverrides(defaultCfg *config.Config, overrides *config.Config) {
	defaultVal := reflect.ValueOf(defaultCfg).Elem()
	overrideVal := reflect.ValueOf(overrides).Elem()

	for i := 0; i < defaultVal.NumField(); i++ {
		defaultField := defaultVal.Field(i)
		overrideField := overrideVal.Field(i)

		if overrideField.IsZero() {
			continue // Skip fields that are not set
		}

		if defaultField.CanSet() {
			defaultField.Set(overrideField)
		}
	}
}

// PurgeAllCollections drops all collections in the specified database.
func PurgeAllCollections(ctx context.Context, client *mongo.Client, databaseName string) error {
	database := client.Database(databaseName)
	collections, err := database.ListCollectionNames(ctx, bson.D{{}})
	if err != nil {
		return err
	}

	for _, collection := range collections {
		if err := database.Collection(collection).Drop(ctx); err != nil {
			return err
		}
	}
	return nil
}

// setupTestDB connects to MongoDB and purges all collections.
func setupTestDB(cfg *config.Config) {
	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cfg.Db.Address))
	if err != nil {
		log.Fatal(err)
	}

	// Purge all collections in the test database
	if err := PurgeAllCollections(context.TODO(), client, cfg.Db.DbName); err != nil {
		log.Fatal("Failed to purge database:", err)
	}
}

func setUpTestQueue(t *testing.T, cfg *queueconfig.QueueConfig) (*queue.QueueManager, *amqp091.Connection, error) {
	amqpURI := fmt.Sprintf("amqp://%s:%s@%s", cfg.QueueUser, cfg.QueuePassword, cfg.Url)
	conn, err := amqp091.Dial(amqpURI)
	if err != nil {
		t.Fatalf("failed to connect to RabbitMQ in test: %v", err)
	}
	err = purgeQueues(conn, []string{
		client.ExpiredStakingQueueName,
		// purge the delay queue as well
		client.ExpiredStakingQueueName + "_delay",
	})
	if err != nil {
		log.Fatal("failed to purge queues in test: ", err)
		return nil, nil, err
	}

	qm, err := queue.NewQueueManager(cfg)
	if err != nil {
		t.Fatalf("failed to setup queue manager in test: %v", err)
	}

	return qm, conn, nil
}

// purgeQueues purges all messages from the given list of queues.
func purgeQueues(conn *amqp091.Connection, queues []string) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel in test: %w", err)
	}
	defer ch.Close()

	for _, queue := range queues {
		_, err := ch.QueuePurge(queue, false)
		if err != nil {
			if strings.Contains(err.Error(), "NOT_FOUND") || strings.Contains(err.Error(), "channel/connection is not open") {
				fmt.Printf("Queue '%s' not found, ignoring...\n", queue)
				continue
			}
			return fmt.Errorf("failed to purge queue in test %s: %w", queue, err)
		}
	}

	return nil
}

func insertTestDelegations(t *testing.T, docs []model.TimeLockDocument) {
	cfg, err := config.New("./config-test.yml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cfg.Db.Address))
	if err != nil {
		log.Fatal(err)
	}
	database := client.Database(cfg.Db.DbName)
	collection := database.Collection(model.TimeLockCollection)

	// Convert slice of TimeLockDocument to slice of interface{} for InsertMany
	var documents []interface{}
	for _, doc := range docs {
		documents = append(documents, doc)
	}

	_, err = collection.InsertMany(context.Background(), documents)
	if err != nil {
		t.Fatalf("Failed to insert test delegations: %v", err)
	}
}

func fetchAllTestDelegations(t *testing.T) []model.TimeLockDocument {
	cfg, err := config.New("./config-test.yml")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(cfg.Db.Address))
	if err != nil {
		log.Fatal(err)
	}
	database := client.Database(cfg.Db.DbName)
	collection := database.Collection(model.TimeLockCollection)

	cursor, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("Failed to fetch test delegations: %v", err)
	}
	defer cursor.Close(context.Background())

	var results []model.TimeLockDocument
	for cursor.Next(context.Background()) {
		var result model.TimeLockDocument
		err := cursor.Decode(&result)
		if err != nil {
			t.Fatalf("Failed to decode test delegations: %v", err)
		}
		results = append(results, result)
	}

	return results
}

// inspectQueueMessageCount inspects the number of messages in the given queue.
func inspectQueueMessageCount(t *testing.T, conn *amqp091.Connection, queueName string) (int, error) {
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open a channel in test: %v", err)
	}
	q, err := ch.QueueDeclarePassive(queueName, false, false, false, false, nil)
	if err != nil {
		if strings.Contains(err.Error(), "NOT_FOUND") || strings.Contains(err.Error(), "channel/connection is not open") {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to inspect queue in test %s: %w", queueName, err)
	}
	return q.Messages, nil
}
