package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Balance struct {
	BalanceID uuid.UUID
	Amount    decimal.Decimal
}
