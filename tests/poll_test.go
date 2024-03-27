package tests

import (
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

func TestProcessExpiredDelegations(t *testing.T) {
	mockDB := new(mocks.DbInterface)
	mockBtc := new(mocks.BtcInterface)

	expectedBtcTip := int64(1000)
	mockBtc.On("GetBlockCount").Return(expectedBtcTip, nil)

	expiredDelegationsFirstCall := []model.StakingExpiryHeightDocument{
		{
			StakingTxHashHex: mockStakingTxHashHex,
			ExpireBtcHeight:  999,
		},
	}
	var expiredDelegationsSubsequentCalls []model.StakingExpiryHeightDocument

	// Setup the sequence
	mockDB.On("FindExpiredDelegations", mock.Anything, mock.Anything).
		Return(expiredDelegationsFirstCall, nil).Once() // Return non-empty slice on first call
	mockDB.On("FindExpiredDelegations", mock.Anything, mock.Anything).
		Return(expiredDelegationsSubsequentCalls, nil).Maybe() // Return empty slice on subsequent calls

	// Assuming you're deleting the delegation after processing
	mockDB.On("DeleteExpiredDelegation", mock.Anything, mockStakingTxHashHex).Return(nil).Once()

	// Integration with test server setup
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
		}, 2*time.Minute, 100*time.Millisecond,
	)
}
