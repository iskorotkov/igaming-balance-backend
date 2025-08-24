package transform

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	balancev1 "github.com/iskorotkov/igaming-balance-backend/gen/balance/v1"
	"github.com/iskorotkov/igaming-balance-backend/internal/db"
	"github.com/iskorotkov/igaming-balance-backend/internal/domain"
	"github.com/shopspring/decimal"
)

var ErrInvalidBalanceID = errors.New("invalid balance id")

func BalanceToProto(b domain.Balance) (*balancev1.BalanceResponse, error) {
	return &balancev1.BalanceResponse{
		BalanceId: b.BalanceID.String(),
		Amount: &balancev1.Decimal{
			Value: b.Amount.String(),
		},
	}, nil
}

func BalanceFromProto(proto *balancev1.BalanceResponse) (domain.Balance, error) {
	balanceID, err := uuid.Parse(proto.GetBalanceId())
	if err != nil {
		return domain.Balance{}, fmt.Errorf("%w: %v", ErrInvalidBalanceID, err)
	}

	amount, err := decimal.NewFromString(proto.Amount.Value)
	if err != nil {
		return domain.Balance{}, fmt.Errorf("%w: %v", ErrInvalidAmount, err)
	}

	return domain.Balance{
		BalanceID: balanceID,
		Amount:    amount,
	}, nil
}

func BalanceFromPgx(b db.Balance) (domain.Balance, error) {
	return domain.Balance{
		BalanceID: b.BalanceID,
		Amount:    b.Amount,
	}, nil
}
