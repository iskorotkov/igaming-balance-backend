package transform_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iskorotkov/igaming-balance-backend/gen/balance/v1"
	"github.com/iskorotkov/igaming-balance-backend/internal/domain"
	"github.com/iskorotkov/igaming-balance-backend/internal/transform"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTxFromProto(t *testing.T) {
	balanceID := uuid.New()
	txID := uuid.New()
	amount := decimal.NewFromInt(100)

	tests := []struct {
		name    string
		proto   *balancev1.RecordTxRequest
		want    domain.Tx
		wantErr error
	}{
		{
			name: "valid deposit transaction",
			proto: &balancev1.RecordTxRequest{
				BalanceId: balanceID.String(),
				TxId:      txID.String(),
				Amount:    &balancev1.Decimal{Value: amount.String()},
				Source:    balancev1.Source_SOURCE_GAME,
				State:     balancev1.State_STATE_DEPOSIT,
			},
			want: domain.Tx{
				BalanceID: balanceID,
				TxID:      txID,
				Amount:    amount,
				Source:    domain.SourceGame,
				State:     domain.StateDeposit,
			},
		},
		{
			name: "valid withdrawal transaction",
			proto: &balancev1.RecordTxRequest{
				BalanceId: balanceID.String(),
				TxId:      txID.String(),
				Amount:    &balancev1.Decimal{Value: amount.String()},
				Source:    balancev1.Source_SOURCE_PAYMENT,
				State:     balancev1.State_STATE_WITHDRAW,
			},
			want: domain.Tx{
				BalanceID: balanceID,
				TxID:      txID,
				Amount:    amount,
				Source:    domain.SourcePayment,
				State:     domain.StateWithdraw,
			},
		},
		{
			name: "invalid balance ID",
			proto: &balancev1.RecordTxRequest{
				BalanceId: "invalid-uuid",
				TxId:      txID.String(),
				Amount:    &balancev1.Decimal{Value: amount.String()},
				Source:    balancev1.Source_SOURCE_GAME,
				State:     balancev1.State_STATE_DEPOSIT,
			},
			wantErr: transform.ErrInvalidBalanceID,
		},
		{
			name: "invalid transaction ID",
			proto: &balancev1.RecordTxRequest{
				BalanceId: balanceID.String(),
				TxId:      "invalid-uuid",
				Amount:    &balancev1.Decimal{Value: amount.String()},
				Source:    balancev1.Source_SOURCE_GAME,
				State:     balancev1.State_STATE_DEPOSIT,
			},
			wantErr: transform.ErrInvalidTxID,
		},
		{
			name: "invalid amount",
			proto: &balancev1.RecordTxRequest{
				BalanceId: balanceID.String(),
				TxId:      txID.String(),
				Amount:    &balancev1.Decimal{Value: "invalid-amount"},
				Source:    balancev1.Source_SOURCE_GAME,
				State:     balancev1.State_STATE_DEPOSIT,
			},
			wantErr: transform.ErrInvalidAmount,
		},
		{
			name: "unspecified source",
			proto: &balancev1.RecordTxRequest{
				BalanceId: balanceID.String(),
				TxId:      txID.String(),
				Amount:    &balancev1.Decimal{Value: amount.String()},
				Source:    balancev1.Source_SOURCE_UNSPECIFIED,
				State:     balancev1.State_STATE_DEPOSIT,
			},
			wantErr: transform.ErrInvalidSource,
		},
		{
			name: "unspecified state",
			proto: &balancev1.RecordTxRequest{
				BalanceId: balanceID.String(),
				TxId:      txID.String(),
				Amount:    &balancev1.Decimal{Value: amount.String()},
				Source:    balancev1.Source_SOURCE_GAME,
				State:     balancev1.State_STATE_UNSPECIFIED,
			},
			wantErr: transform.ErrInvalidState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transform.TxFromProto(tt.proto)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.BalanceID, got.BalanceID)
			assert.Equal(t, tt.want.TxID, got.TxID)
			assert.True(t, tt.want.Amount.Equal(got.Amount))
			assert.Equal(t, tt.want.Source, got.Source)
			assert.Equal(t, tt.want.State, got.State)
		})
	}
}

func TestTxToProto(t *testing.T) {
	balanceID := uuid.New()
	txID := uuid.New()
	amount := decimal.NewFromInt(100)
	createdAt := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name string
		tx   domain.Tx
		want *balancev1.Tx
	}{
		{
			name: "valid deposit transaction",
			tx: domain.Tx{
				BalanceID: balanceID,
				TxID:      txID,
				Amount:    amount,
				Source:    domain.SourceGame,
				State:     domain.StateDeposit,
				CreatedAt: createdAt,
			},
			want: &balancev1.Tx{
				BalanceId: balanceID.String(),
				TxId:      txID.String(),
				Amount:    &balancev1.Decimal{Value: amount.String()},
				Source:    balancev1.Source_SOURCE_GAME,
				State:     balancev1.State_STATE_DEPOSIT,
				CreatedAt: timestamppb.New(createdAt),
			},
		},
		{
			name: "valid withdrawal transaction",
			tx: domain.Tx{
				BalanceID: balanceID,
				TxID:      txID,
				Amount:    amount,
				Source:    domain.SourcePayment,
				State:     domain.StateWithdraw,
				CreatedAt: createdAt,
			},
			want: &balancev1.Tx{
				BalanceId: balanceID.String(),
				TxId:      txID.String(),
				Amount:    &balancev1.Decimal{Value: amount.String()},
				Source:    balancev1.Source_SOURCE_PAYMENT,
				State:     balancev1.State_STATE_WITHDRAW,
				CreatedAt: timestamppb.New(createdAt),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transform.TxToProto(tt.tx)

			require.NoError(t, err)
			assert.Equal(t, tt.want.BalanceId, got.BalanceId)
			assert.Equal(t, tt.want.TxId, got.TxId)
			assert.Equal(t, tt.want.Amount, got.Amount)
			assert.Equal(t, tt.want.Source, got.Source)
			assert.Equal(t, tt.want.State, got.State)
			assert.Equal(t, tt.want.CreatedAt.AsTime(), got.CreatedAt.AsTime())
		})
	}
}

func TestBalanceFromProto(t *testing.T) {
	balanceID := uuid.New()
	amount := decimal.NewFromInt(1000)

	tests := []struct {
		name    string
		proto   *balancev1.BalanceResponse
		want    domain.Balance
		wantErr error
	}{
		{
			name: "valid balance",
			proto: &balancev1.BalanceResponse{
				BalanceId: balanceID.String(),
				Amount:    &balancev1.Decimal{Value: amount.String()},
			},
			want: domain.Balance{
				BalanceID: balanceID,
				Amount:    amount,
			},
		},
		{
			name: "invalid balance ID",
			proto: &balancev1.BalanceResponse{
				BalanceId: "invalid-uuid",
				Amount:    &balancev1.Decimal{Value: amount.String()},
			},
			wantErr: transform.ErrInvalidBalanceID,
		},
		{
			name: "invalid amount",
			proto: &balancev1.BalanceResponse{
				BalanceId: balanceID.String(),
				Amount:    &balancev1.Decimal{Value: "invalid-amount"},
			},
			wantErr: transform.ErrInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transform.BalanceFromProto(tt.proto)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.BalanceID, got.BalanceID)
			assert.True(t, tt.want.Amount.Equal(got.Amount))
		})
	}
}

func TestBalanceToProto(t *testing.T) {
	balanceID := uuid.New()
	amount := decimal.NewFromInt(1000)

	tests := []struct {
		name string
		bal  domain.Balance
		want *balancev1.BalanceResponse
	}{
		{
			name: "valid balance",
			bal: domain.Balance{
				BalanceID: balanceID,
				Amount:    amount,
			},
			want: &balancev1.BalanceResponse{
				BalanceId: balanceID.String(),
				Amount:    &balancev1.Decimal{Value: amount.String()},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transform.BalanceToProto(tt.bal)

			require.NoError(t, err)
			assert.Equal(t, tt.want.BalanceId, got.BalanceId)
			assert.Equal(t, tt.want.Amount, got.Amount)
		})
	}
}
