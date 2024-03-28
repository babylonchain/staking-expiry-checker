package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/babylonchain/staking-expiry-checker/internal/db/model"
	"github.com/babylonchain/staking-expiry-checker/tests/mocks"
)

const (
	mockStakingTxHashHex = "0x1234567890abcdef"
)

func TestProcessExpiredDelegations_NoErrors(t *testing.T) {
	mockDB := new(mocks.DbInterface)
	mockBtc := new(mocks.BtcInterface)

	expectedBtcTip := int64(1000)
	mockBtc.On("GetBlockCount").Return(expectedBtcTip, nil)

	expiredDelegationsFirstCall := []model.TimeLockDocument{
		{
			StakingTxHashHex: mockStakingTxHashHex,
			ExpireHeight:     999,
		},
	}
	var expiredDelegationsSubsequentCalls []model.TimeLockDocument

	mockDB.On("FindExpiredDelegations", mock.Anything, mock.Anything).
		Return(expiredDelegationsFirstCall, nil).Once() // Return non-empty slice on first call
	mockDB.On("FindExpiredDelegations", mock.Anything, mock.Anything).
		Return(expiredDelegationsSubsequentCalls, nil).Maybe() // Return empty slice on subsequent calls

	mockDB.On("DeleteExpiredDelegation", mock.Anything, mockStakingTxHashHex).Return(nil).Once()

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

	mockDB.On("FindExpiredDelegations", mock.Anything, expectedBtcTip).
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

	expiredDelegation := model.TimeLockDocument{StakingTxHashHex: "someHash", ExpireHeight: 999}

	mockDB.On("FindExpiredDelegations", mock.Anything, expectedBtcTip).
		Return([]model.TimeLockDocument{expiredDelegation}, nil)
	mockDB.On("DeleteExpiredDelegation", mock.Anything, "someHash").
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
