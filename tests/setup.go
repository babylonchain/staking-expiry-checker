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
	"github.com/babylonchain/staking-expiry-checker/internal/observability/metrics"
	"github.com/babylonchain/staking-expiry-checker/internal/poller"
	"github.com/babylonchain/staking-expiry-checker/internal/queue"
	"github.com/babylonchain/staking-expiry-checker/internal/queue/client"
	"github.com/babylonchain/staking-expiry-checker/internal/services"
)

type TestServerDependency struct {
	ConfigOverrides *config.Config
	MockDbClient    db.DbInterface
	MockBtcClient   btcclient.BtcInterface
}

func setupTestServer(t *testing.T, dep *TestServerDependency) (*queue.QueueManager, func()) {
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

	qm, err := setUpTestQueue(t, &cfg.Queue)
	if err != nil {
		t.Fatalf("Failed to setup test queue: %v", err)
	}

	var (
		dbClient  db.DbInterface
		btcClient btcclient.BtcInterface
	)

	if dep != nil && dep.MockBtcClient != nil && dep.MockDbClient != nil {
		dbClient = dep.MockDbClient
		btcClient = dep.MockBtcClient
	} else {
		setupTestDB(cfg)
		dbClient, err = db.New(ctx, cfg.Db.DbName, cfg.Db.Address)
		if err != nil {
			t.Fatalf("Failed to initialize db client: %v", err)
		}

		btcClient, err = btcclient.NewBtcClient(&cfg.Btc)
		if err != nil {
			t.Fatalf("Failed to initialize btc client: %v", err)
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
		cancel() // Cancel the context to release resources
	}

	go p.Start(ctx)
	return qm, teardown
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

func setUpTestQueue(t *testing.T, cfg *config.QueueConfig) (*queue.QueueManager, error) {
	amqpURI := fmt.Sprintf("amqp://%s:%s@%s", cfg.User, cfg.Pass, cfg.Url)
	conn, err := amqp091.Dial(amqpURI)
	if err != nil {
		t.Fatalf("failed to connect to RabbitMQ in test: %v", err)
	}
	defer conn.Close()
	err = purgeQueues(conn, []string{
		client.ExpiredStakingQueueName,
	})
	if err != nil {
		log.Fatal("failed to purge queues in test: ", err)
		return nil, err
	}

	qm, err := queue.NewQueueManager(cfg)
	if err != nil {
		t.Fatalf("failed to setup queue manager in test: %v", err)
	}

	return qm, nil
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
			if strings.Contains(err.Error(), "no queue") {
				fmt.Printf("Queue '%s' not found, ignoring...\n", queue)
				continue // Ignore this error and proceed with the next queue
			}
			return fmt.Errorf("failed to purge queue in test %s: %w", queue, err)
		}
	}

	return nil
}
