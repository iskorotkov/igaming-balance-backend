package service

import (
	"context"
	"errors"
	"testing"

	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	balancev1 "github.com/iskorotkov/igaming-balance-backend/gen/balance/v1"
	"github.com/iskorotkov/igaming-balance-backend/internal/domain"
	"github.com/iskorotkov/igaming-balance-backend/internal/storage"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestBalances_ListTx(t *testing.T) {
	balanceID := uuid.New()
	txID := uuid.New()
	amount := decimal.NewFromInt(100)
	createdAt := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name           string
		request        *balancev1.ListTxRequest
		setupMock      func(*MockStorage)
		expectedStatus connect.Code
		expectedTxs    int
	}{
		{
			name: "list recent transactions success",
			request: &balancev1.ListTxRequest{
				BalanceId: balanceID.String(),
				PageSize:  10,
			},
			setupMock: func(m *MockStorage) {
				txs := []domain.Tx{
					{
						BalanceID: balanceID,
						TxID:      txID,
						Amount:    amount,
						Source:    domain.SourcePayment,
						State:     domain.StateDeposit,
						CreatedAt: createdAt,
					},
				}
				m.EXPECT().RecentTxs(context.Background(), balanceID, false, 10).Return(txs, nil)
			},
			expectedTxs: 1,
		},
		{
			name: "invalid balance ID",
			request: &balancev1.ListTxRequest{
				BalanceId: "invalid-uuid",
				PageSize:  10,
			},
			setupMock:      func(m *MockStorage) {},
			expectedStatus: connect.CodeInvalidArgument,
		},
		{
			name: "storage error",
			request: &balancev1.ListTxRequest{
				BalanceId: balanceID.String(),
				PageSize:  10,
			},
			setupMock: func(m *MockStorage) {
				m.EXPECT().RecentTxs(context.Background(), balanceID, false, 10).Return([]domain.Tx{}, errors.New("storage error"))
			},
			expectedStatus: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewMockStorage(t)
			tt.setupMock(mockStorage)

			service := NewBalances(mockStorage)
			ctx := context.Background()

			resp, err := service.ListTx(ctx, connect.NewRequest(tt.request))

			if tt.expectedStatus != 0 {
				require.Error(t, err)
				connectErr := err.(*connect.Error)
				assert.Equal(t, tt.expectedStatus, connectErr.Code())
				return
			}

			require.NoError(t, err)
			assert.Len(t, resp.Msg.Txs, tt.expectedTxs)
		})
	}
}

func TestBalances_RecordTx(t *testing.T) {
	balanceID := uuid.New()
	txID := uuid.New()
	amount := decimal.NewFromInt(100)

	tests := []struct {
		name           string
		request        *balancev1.RecordTxRequest
		setupMock      func(*MockStorage)
		expectedStatus connect.Code
	}{
		{
			name: "record transaction success",
			request: &balancev1.RecordTxRequest{
				BalanceId: balanceID.String(),
				TxId:      txID.String(),
				Amount:    &balancev1.Decimal{Value: amount.String()},
				Source:    balancev1.Source_SOURCE_PAYMENT,
				State:     balancev1.State_STATE_DEPOSIT,
			},
			setupMock: func(m *MockStorage) {
				tx := domain.Tx{
					BalanceID: balanceID,
					TxID:      txID,
					Amount:    amount,
					Source:    domain.SourcePayment,
					State:     domain.StateDeposit,
				}
				m.EXPECT().RecordTx(context.Background(), tx).Return(nil)
			},
		},
		{
			name: "balance not found",
			request: &balancev1.RecordTxRequest{
				BalanceId: balanceID.String(),
				TxId:      txID.String(),
				Amount:    &balancev1.Decimal{Value: amount.String()},
				Source:    balancev1.Source_SOURCE_PAYMENT,
				State:     balancev1.State_STATE_DEPOSIT,
			},
			setupMock: func(m *MockStorage) {
				tx := domain.Tx{
					BalanceID: balanceID,
					TxID:      txID,
					Amount:    amount,
					Source:    domain.SourcePayment,
					State:     domain.StateDeposit,
				}
				m.EXPECT().RecordTx(context.Background(), tx).Return(storage.ErrNotFound)
			},
			expectedStatus: connect.CodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewMockStorage(t)
			tt.setupMock(mockStorage)

			service := NewBalances(mockStorage)
			ctx := context.Background()

			resp, err := service.RecordTx(ctx, connect.NewRequest(tt.request))

			if tt.expectedStatus != 0 {
				require.Error(t, err)
				connectErr := err.(*connect.Error)
				assert.Equal(t, tt.expectedStatus, connectErr.Code())
				return
			}

			require.NoError(t, err)
			assert.IsType(t, &connect.Response[emptypb.Empty]{}, resp)
		})
	}
}

func TestBalances_OpenBalance(t *testing.T) {
	balanceID := uuid.New()

	tests := []struct {
		name           string
		request        *balancev1.OpenBalanceRequest
		setupMock      func(*MockStorage)
		expectedStatus connect.Code
	}{
		{
			name: "open balance success",
			request: &balancev1.OpenBalanceRequest{
				BalanceId: balanceID.String(),
			},
			setupMock: func(m *MockStorage) {
				m.EXPECT().OpenBalance(context.Background(), balanceID).Return(nil)
			},
		},
		{
			name: "invalid balance ID",
			request: &balancev1.OpenBalanceRequest{
				BalanceId: "invalid-uuid",
			},
			setupMock:      func(m *MockStorage) {},
			expectedStatus: connect.CodeInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewMockStorage(t)
			tt.setupMock(mockStorage)

			service := NewBalances(mockStorage)
			ctx := context.Background()

			resp, err := service.OpenBalance(ctx, connect.NewRequest(tt.request))

			if tt.expectedStatus != 0 {
				require.Error(t, err)
				connectErr := err.(*connect.Error)
				assert.Equal(t, tt.expectedStatus, connectErr.Code())
				return
			}

			require.NoError(t, err)
			assert.IsType(t, &connect.Response[emptypb.Empty]{}, resp)
		})
	}
}

func TestBalances_Balance(t *testing.T) {
	balanceID := uuid.New()
	amount := decimal.NewFromInt(1000)

	tests := []struct {
		name           string
		request        *balancev1.BalanceRequest
		setupMock      func(*MockStorage)
		expectedStatus connect.Code
	}{
		{
			name: "get balance success",
			request: &balancev1.BalanceRequest{
				BalanceId: balanceID.String(),
			},
			setupMock: func(m *MockStorage) {
				balance := domain.Balance{
					BalanceID: balanceID,
					Amount:    amount,
				}
				m.EXPECT().Balance(context.Background(), balanceID).Return(balance, nil)
			},
		},
		{
			name: "balance not found",
			request: &balancev1.BalanceRequest{
				BalanceId: balanceID.String(),
			},
			setupMock: func(m *MockStorage) {
				m.EXPECT().Balance(context.Background(), balanceID).Return(domain.Balance{}, storage.ErrNotFound)
			},
			expectedStatus: connect.CodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewMockStorage(t)
			tt.setupMock(mockStorage)

			service := NewBalances(mockStorage)
			ctx := context.Background()

			resp, err := service.Balance(ctx, connect.NewRequest(tt.request))

			if tt.expectedStatus != 0 {
				require.Error(t, err)
				connectErr := err.(*connect.Error)
				assert.Equal(t, tt.expectedStatus, connectErr.Code())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, balanceID.String(), resp.Msg.BalanceId)
			assert.Equal(t, amount.String(), resp.Msg.Amount.Value)
		})
	}
}

func TestBalances_CancelTxs(t *testing.T) {
	balanceID := uuid.New()
	txID1 := uuid.New()
	txID2 := uuid.New()

	tests := []struct {
		name           string
		request        *balancev1.CancelTxsRequest
		setupMock      func(*MockStorage)
		expectedStatus connect.Code
	}{
		{
			name: "cancel transactions success",
			request: &balancev1.CancelTxsRequest{
				BalanceId: balanceID.String(),
				TxIds:     []string{txID1.String(), txID2.String()},
			},
			setupMock: func(m *MockStorage) {
				expectedTxIDs := []uuid.UUID{txID1, txID2}
				m.EXPECT().CancelTxs(context.Background(), balanceID, expectedTxIDs).Return(nil)
			},
		},
		{
			name: "no transaction IDs provided",
			request: &balancev1.CancelTxsRequest{
				BalanceId: balanceID.String(),
				TxIds:     []string{},
			},
			setupMock:      func(m *MockStorage) {},
			expectedStatus: connect.CodeInvalidArgument,
		},
		{
			name: "invalid balance ID",
			request: &balancev1.CancelTxsRequest{
				BalanceId: "invalid-uuid",
				TxIds:     []string{txID1.String()},
			},
			setupMock:      func(m *MockStorage) {},
			expectedStatus: connect.CodeInvalidArgument,
		},
		{
			name: "invalid transaction ID",
			request: &balancev1.CancelTxsRequest{
				BalanceId: balanceID.String(),
				TxIds:     []string{"invalid-uuid"},
			},
			setupMock:      func(m *MockStorage) {},
			expectedStatus: connect.CodeInvalidArgument,
		},
		{
			name: "transactions not found",
			request: &balancev1.CancelTxsRequest{
				BalanceId: balanceID.String(),
				TxIds:     []string{txID1.String()},
			},
			setupMock: func(m *MockStorage) {
				expectedTxIDs := []uuid.UUID{txID1}
				m.EXPECT().CancelTxs(context.Background(), balanceID, expectedTxIDs).Return(storage.ErrNotFound)
			},
			expectedStatus: connect.CodeNotFound,
		},
		{
			name: "negative balance error",
			request: &balancev1.CancelTxsRequest{
				BalanceId: balanceID.String(),
				TxIds:     []string{txID1.String()},
			},
			setupMock: func(m *MockStorage) {
				expectedTxIDs := []uuid.UUID{txID1}
				m.EXPECT().CancelTxs(context.Background(), balanceID, expectedTxIDs).Return(storage.ErrNegativeBalance)
			},
			expectedStatus: connect.CodeInvalidArgument,
		},
		{
			name: "storage error",
			request: &balancev1.CancelTxsRequest{
				BalanceId: balanceID.String(),
				TxIds:     []string{txID1.String()},
			},
			setupMock: func(m *MockStorage) {
				expectedTxIDs := []uuid.UUID{txID1}
				m.EXPECT().CancelTxs(context.Background(), balanceID, expectedTxIDs).Return(errors.New("storage error"))
			},
			expectedStatus: connect.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewMockStorage(t)
			tt.setupMock(mockStorage)

			service := NewBalances(mockStorage)
			ctx := context.Background()

			resp, err := service.CancelTxs(ctx, connect.NewRequest(tt.request))

			if tt.expectedStatus != 0 {
				require.Error(t, err)
				connectErr := err.(*connect.Error)
				assert.Equal(t, tt.expectedStatus, connectErr.Code())
				return
			}

			require.NoError(t, err)
			assert.IsType(t, &connect.Response[emptypb.Empty]{}, resp)
		})
	}
}
