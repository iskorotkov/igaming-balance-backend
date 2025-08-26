package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

//go:generate go run github.com/dmarkham/enumer -type=Source -trimprefix=Source -json -text -yaml -sql
//go:generate go run github.com/dmarkham/enumer -type=State -trimprefix=State -json -text -yaml -sql

const (
	SourceUnknown Source = iota
	SourceGame
	SourcePayment
	SourceService
)

type Source int

const (
	StateUnknown State = iota
	StateDeposit
	StateWithdraw
)

type State int

type Tx struct {
	CreatedAt time.Time
	DeletedAt *time.Time // Use soft deletes.
	TxID      uuid.UUID
	BalanceID uuid.UUID
	Source    Source
	State     State
	Amount    decimal.Decimal
}
