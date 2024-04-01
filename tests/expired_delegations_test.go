package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/babylonchain/staking-expiry-checker/internal/db/model"
	"github.com/babylonchain/staking-expiry-checker/internal/types"
	"github.com/babylonchain/staking-expiry-checker/tests/mocks"
)

func TestProcessExpiredDelegations_NoErrors(t *testing.T) {
	mockDB := new(mocks.DbInterface)
	mockBtc := new(mocks.BtcInterface)

	expectedBtcTip := int64(1000)
	mockBtc.On("GetBlockCount").Return(expectedBtcTip, nil)

	// Create an ObjectID for testing purposes
	testID, _ := primitive.ObjectIDFromHex("507f1f77bcf86cd799439011")
	expiredDelegationsFirstCall := []model.TimeLockDocument{
		{
			ID:               testID,
			StakingTxHashHex: "mockStakingTxHashHex",
			ExpireHeight:     999,
			TxType:           types.Active,
		},
	}
	var expiredDelegationsSubsequentCalls []model.TimeLockDocument

	mockDB.On("FindExpiredDelegations", mock.Anything, uint64(expectedBtcTip)).
		Return(expiredDelegationsFirstCall, nil).Once() // Return non-empty slice on first call
	mockDB.On("FindExpiredDelegations", mock.Anything, uint64(expectedBtcTip)).
		Return(expiredDelegationsSubsequentCalls, nil).Maybe() // Return empty slice on subsequent calls

	mockDB.On("DeleteExpiredDelegation", mock.Anything, expiredDelegationsFirstCall[0].ID).Return(nil).Once()

	qm, teardown := setupTestServer(t, &TestServerDependency{
		MockDbClient:  mockDB,
		MockBtcClient: mockBtc,
	})
	defer teardown()

	// Wait for the data
	require.Eventually(
		t, func() bool {
			expiredQueueMessageCount, err := qm.GetExpiredQueueMessageCount()
			return err == nil && expiredQueueMessageCount == 1
		}, 10*time.Second, 100*time.Millisecond,
	)
}

func TestProcessExpiredDelegations_ErrorGettingBlockCount(t *testing.T) {
	mockDB := new(mocks.DbInterface)
	mockBtc := new(mocks.BtcInterface)

	mockBtc.On("GetBlockCount").Return(int64(0), errors.New("failed to get block count"))

	qm, teardown := setupTestServer(t, &TestServerDependency{
		MockDbClient:  mockDB,
		MockBtcClient: mockBtc,
	})
	defer teardown()

	// Verify the process handles the error as expected
	require.Eventually(
		t, func() bool {
			expiredQueueMessageCount, err := qm.GetExpiredQueueMessageCount()
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

	qm, teardown := setupTestServer(t, &TestServerDependency{
		MockDbClient:  mockDB,
		MockBtcClient: mockBtc,
	})
	defer teardown()

	// Verify the process handles the error as expected
	require.Eventually(
		t, func() bool {
			expiredQueueMessageCount, err := qm.GetExpiredQueueMessageCount()
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
		TxType:           types.Active,
	}

	mockDB.On("FindExpiredDelegations", mock.Anything, uint64(expectedBtcTip)).
		Return([]model.TimeLockDocument{expiredDelegation}, nil)
	mockDB.On("DeleteExpiredDelegation", mock.Anything, testID).
		Return(errors.New("delete error"))

	qm, teardown := setupTestServer(t, &TestServerDependency{
		MockDbClient:  mockDB,
		MockBtcClient: mockBtc,
	})
	defer teardown()

	// Verify the process handles the error as expected
	require.Eventually(
		t, func() bool {
			expiredQueueMessageCount, err := qm.GetExpiredQueueMessageCount()
			return err == nil && expiredQueueMessageCount == 0
		}, 10*time.Second, 100*time.Millisecond,
	)
}
