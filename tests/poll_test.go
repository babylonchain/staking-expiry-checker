package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

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

	expiredDelegations := []model.StakingExpiryHeightDocument{
		{
			StakingTxHashHex: mockStakingTxHashHex,
			ExpireBtcHeight:  999,
		},
	}
	mockDB.On("FindExpiredDelegations", mock.Anything, mock.Anything).Return(expiredDelegations, nil)

	// Integration with test server setup
	_, teardown := setupTestServer(t, &TestServerDependency{
		MockDbClient:  mockDB,
		MockBtcClient: mockBtc,
	})
	defer teardown()

	//mockDB.AssertExpectations(t)
	//mockBtc.AssertExpectations(t)

	time.Sleep(10 * time.Second)

	//expiredQueueMessageCount, err := queueManager.GetExpiredQueueMessageCount()
	//assert.NoError(t, err)
	//assert.Equal(t, 1, expiredQueueMessageCount)
}
