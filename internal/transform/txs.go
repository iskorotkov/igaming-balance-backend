package transform

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	balancev1 "github.com/iskorotkov/igaming-balance-backend/gen/balance/v1"
	"github.com/iskorotkov/igaming-balance-backend/internal/db"
	"github.com/iskorotkov/igaming-balance-backend/internal/domain"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrInvalidTxID   = errors.New("invalid tx id")
	ErrInvalidSource = errors.New("invalid source")
	ErrInvalidState  = errors.New("invalid state")
	ErrInvalidAmount = errors.New("invalid amount")
)

func TxFromProto(tx *balancev1.RecordTxRequest) (domain.Tx, error) {
	if tx.GetSource() == balancev1.Source_SOURCE_UNSPECIFIED {
		return domain.Tx{}, fmt.Errorf("%w: %v", ErrInvalidSource, "source is unspecified")
	}

	if tx.GetState() == balancev1.State_STATE_UNSPECIFIED {
		return domain.Tx{}, fmt.Errorf("%w: %v", ErrInvalidState, "state is unspecified")
	}

	balanceID, err := uuid.Parse(tx.GetBalanceId())
	if err != nil {
		return domain.Tx{}, fmt.Errorf("%w: %v", ErrInvalidBalanceID, err)
	}

	txID, err := uuid.Parse(tx.GetTxId())
	if err != nil {
		return domain.Tx{}, fmt.Errorf("%w: %v", ErrInvalidTxID, err)
	}

	amount, err := decimal.NewFromString(tx.Amount.Value)
	if err != nil {
		return domain.Tx{}, fmt.Errorf("%w: %v", ErrInvalidAmount, err)
	}

	return domain.Tx{
		TxID:      txID,
		BalanceID: balanceID,
		Source:    domain.Source(tx.GetSource()),
		State:     domain.State(tx.GetState()),
		Amount:    amount,
	}, nil
}

func TxToProto(tx domain.Tx) (*balancev1.Tx, error) {
	var deletedAt *timestamppb.Timestamp
	if tx.DeletedAt != nil {
		deletedAt = timestamppb.New(*tx.DeletedAt)
	}

	return &balancev1.Tx{
		CreatedAt: timestamppb.New(tx.CreatedAt),
		DeletedAt: deletedAt,
		TxId:      tx.TxID.String(),
		BalanceId: tx.BalanceID.String(),
		Source:    balancev1.Source(tx.Source),
		State:     balancev1.State(tx.State),
		Amount: &balancev1.Decimal{
			Value: tx.Amount.String(),
		},
	}, nil
}

func TxFromPgx(tx db.Tx) (domain.Tx, error) {
	return domain.Tx{
		CreatedAt: tx.CreatedAt,
		DeletedAt: tx.DeletedAt,
		TxID:      tx.TxID,
		BalanceID: tx.BalanceID,
		Source:    tx.Source,
		State:     tx.State,
		Amount:    tx.Amount,
	}, nil
}

func TxToPgx(tx domain.Tx) (db.InsertTxParams, error) {
	return db.InsertTxParams{
		TxID:      tx.TxID,
		BalanceID: tx.BalanceID,
		Source:    tx.Source,
		State:     tx.State,
		Amount:    tx.Amount,
	}, nil
}
