package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/babylonchain/staking-expiry-checker/internal/db/model"
	"github.com/babylonchain/staking-expiry-checker/tests/mocks"
	"github.com/babylonchain/staking-queue-client/client"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestProcessExpiredDelegations_NoErrors(t *testing.T) {
	// setup mock btc client
	mockBtc := new(mocks.BtcInterface)
	expectedBtcTip := int64(1000)
	mockBtc.On("GetBlockCount").Return(expectedBtcTip, nil)

	// assert that db is empty
	docs := fetchAllTestDelegations(t)
	require.Empty(t, docs)

	// setup test server
	_, conn, teardown := setupTestServer(t, &TestServerDependency{
		MockBtcClient: mockBtc,
	})
	defer teardown()

	// insert in db
	expiredDelegations := []model.TimeLockDocument{
		{
			ID:               primitive.NewObjectID(),
			StakingTxHashHex: "mockStakingTxHashHex1",
			ExpireHeight:     999,
			TxType:           "active",
		},

		{
			ID:               primitive.NewObjectID(),
			StakingTxHashHex: "mockStakingTxHashHex2",
			ExpireHeight:     999,
			TxType:           "unbonding",
		},
	}
	insertTestDelegations(t, expiredDelegations)

	// Wait for the data
	require.Eventually(
		t, func() bool {
			expiredQueueMessageCount, err := inspectQueueMessageCount(t, conn, client.ExpiredStakingQueueName)
			return err == nil && expiredQueueMessageCount == 2
		}, 10*time.Second, 100*time.Millisecond,
	)

	// TODO: assert message contents to ensure the correct data is being sent

	// assert that documents are deleted now and db is empty
	docs = fetchAllTestDelegations(t)
	require.Empty(t, docs)
}

func TestProcessExpiredDelegations_ErrorGettingBlockCount(t *testing.T) {
	mockDB := new(mocks.DbInterface)
	mockBtc := new(mocks.BtcInterface)

	mockBtc.On("GetBlockCount").Return(int64(0), errors.New("failed to get block count"))

	_, conn, teardown := setupTestServer(t, &TestServerDependency{
		MockDbClient:  mockDB,
		MockBtcClient: mockBtc,
	})
	defer teardown()

	// Verify the process handles the error as expected
	require.Eventually(
		t, func() bool {
			expiredQueueMessageCount, err := inspectQueueMessageCount(t, conn, client.ExpiredStakingQueueName)
			return err == nil && expiredQueueMessageCount == 0
		}, 10*time.Second, 100*time.Millisecond,
	)
}

func TestProcessExpiredDelegations_ErrorFindingExpiredDelegations(t *testing.T) {
	mockDB := new(mocks.DbInterface)
	mockBtc := new(mocks.BtcInterface)

	expectedBtcTip := int64(1000)
	mockBtc.On("GetBlockCount").Return(expectedBtcTip, nil)

	mockDB.On("FindExpiredDelegations", mock.Anything, uint64(expectedBtcTip)).
		Return(nil, errors.New("database error"))

	_, conn, teardown := setupTestServer(t, &TestServerDependency{
		MockDbClient:  mockDB,
		MockBtcClient: mockBtc,
	})
	defer teardown()

	// Verify the process handles the error as expected
	require.Eventually(
		t, func() bool {
			expiredQueueMessageCount, err := inspectQueueMessageCount(t, conn, client.ExpiredStakingQueueName)
			return err == nil && expiredQueueMessageCount == 0
		}, 10*time.Second, 100*time.Millisecond,
	)
}

func TestProcessExpiredDelegations_ErrorDeletingExpiredDelegation(t *testing.T) {
	mockDB := new(mocks.DbInterface)
	mockBtc := new(mocks.BtcInterface)

	expectedBtcTip := int64(1000)
	mockBtc.On("GetBlockCount").Return(expectedBtcTip, nil)

	// Create an ObjectID for testing purposes
	testID, _ := primitive.ObjectIDFromHex("507f1f77bcf86cd799439011")
	expiredDelegation := model.TimeLockDocument{
		ID:               testID,
		StakingTxHashHex: "mockStakingTxHashHex",
		ExpireHeight:     999,
		TxType:           "active",
	}

	mockDB.On("FindExpiredDelegations", mock.Anything, uint64(expectedBtcTip)).
		Return([]model.TimeLockDocument{expiredDelegation}, nil)
	mockDB.On("DeleteExpiredDelegation", mock.Anything, testID).
		Return(errors.New("delete error"))

	_, conn, teardown := setupTestServer(t, &TestServerDependency{
		MockDbClient:  mockDB,
		MockBtcClient: mockBtc,
	})
	defer teardown()

	// Verify the process handles the error as expected
	require.Eventually(
		t, func() bool {
			expiredQueueMessageCount, err := inspectQueueMessageCount(t, conn, client.ExpiredStakingQueueName)
			return err == nil && expiredQueueMessageCount == 0
		}, 10*time.Second, 100*time.Millisecond,
	)
}
