package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/iskorotkov/igaming-balance-backend/internal/db"
	"github.com/iskorotkov/igaming-balance-backend/internal/domain"
	"github.com/iskorotkov/igaming-balance-backend/internal/transform"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrNegativeBalance = errors.New("negative balance")
)

type ConnectionPool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Querier interface {
	WithTx(tx pgx.Tx) *db.Queries
	RecentTxs(ctx context.Context, arg db.RecentTxsParams) ([]db.Tx, error)
	PreviousTxs(ctx context.Context, arg db.PreviousTxsParams) ([]db.Tx, error)
	OpenBalance(ctx context.Context, balanceID uuid.UUID) (int64, error)
	Balance(ctx context.Context, balanceID uuid.UUID) (db.Balance, error)
}

func NewBalances(c ConnectionPool, q Querier) *Balances {
	return &Balances{
		c: c,
		q: q,
	}
}

type Balances struct {
	c ConnectionPool
	q Querier
}

func (b *Balances) RecordTx(ctx context.Context, tx domain.Tx) error {
	dbTx, err := transform.TxToPgx(tx)
	if err != nil {
		return fmt.Errorf("transform tx: %w", err)
	}

	balanceChange := tx.Amount
	if tx.State == domain.StateWithdraw {
		balanceChange = balanceChange.Neg()
	}

	pgxTx, err := b.c.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin pgx tx: %w", err)
	}
	defer func() {
		if err := pgxTx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.ErrorContext(ctx, "failed to rollback transaction", "error", err)
		}
	}()

	qtx := b.q.WithTx(pgxTx)

	if _, err := qtx.LockBalance(ctx, tx.BalanceID); err != nil {
		return fmt.Errorf("lock balance: %w", err)
	}

	updated, err := qtx.UpdateBalance(ctx, db.UpdateBalanceParams{
		BalanceID: tx.BalanceID,
		Amount:    balanceChange,
	})
	if err != nil {
		if isPgCode(err, "23514") {
			return fmt.Errorf("%w: %v", ErrNegativeBalance, err)
		}
		return fmt.Errorf("update balance: %w", err)
	}
	if updated == 0 {
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	}

	if _, err := qtx.InsertTx(ctx, dbTx); err != nil {
		if isPgCode(err, "23505") {
			return fmt.Errorf("%w: %v", ErrAlreadyExists, err)
		}
		return fmt.Errorf("insert tx: %w", err)
	}

	if err := pgxTx.Commit(ctx); err != nil {
		return fmt.Errorf("commit pgx tx: %w", err)
	}

	return nil
}

func (b *Balances) CancelTxs(ctx context.Context, balanceID uuid.UUID, txIDs []uuid.UUID) error {
	pgxTx, err := b.c.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin pgx tx: %w", err)
	}
	defer func() {
		if err := pgxTx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.ErrorContext(ctx, "failed to rollback transaction", "error", err)
		}
	}()

	qtx := b.q.WithTx(pgxTx)

	if _, err := qtx.LockBalance(ctx, balanceID); err != nil {
		return fmt.Errorf("lock balance: %w", err)
	}

	txs, err := qtx.TxsByID(ctx, db.TxsByIDParams{
		BalanceID: balanceID,
		TxIds:     txIDs,
	})
	if err != nil {
		return fmt.Errorf("get txs: %w", err)
	}
	if len(txs) == 0 {
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	}

	var balanceChange decimal.Decimal
	for _, tx := range txs {
		switch tx.State {
		case domain.StateDeposit:
			balanceChange = balanceChange.Sub(tx.Amount)
		case domain.StateWithdraw:
			balanceChange = balanceChange.Add(tx.Amount)
		default:
			return fmt.Errorf("unknown state: %v", tx.State)
		}
	}

	updated, err := qtx.UpdateBalance(ctx, db.UpdateBalanceParams{
		BalanceID: balanceID,
		Amount:    balanceChange,
	})
	if err != nil {
		if isPgCode(err, "23514") {
			return fmt.Errorf("%w: %v", ErrNegativeBalance, err)
		}
		return fmt.Errorf("update balance: %w", err)
	}
	if updated == 0 {
		return fmt.Errorf("%w: %v", ErrNotFound, err)
	}

	if _, err := qtx.DeleteTxs(ctx, db.DeleteTxsParams{
		BalanceID: balanceID,
		TxIds:     txIDs,
	}); err != nil {
		return fmt.Errorf("insert tx: %w", err)
	}

	if err := pgxTx.Commit(ctx); err != nil {
		return fmt.Errorf("commit pgx tx: %w", err)
	}

	return nil
}

func (b *Balances) RecentTxs(
	ctx context.Context,
	balanceID uuid.UUID,
	includeDeleted bool,
	limit int,
) ([]domain.Tx, error) {
	rows, err := b.q.RecentTxs(ctx, db.RecentTxsParams{
		BalanceID:      balanceID,
		IncludeDeleted: includeDeleted,
		Limit:          int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("fetch txs: %w", err)
	}

	var txs []domain.Tx
	for _, r := range rows {
		t, err := transform.TxFromPgx(r)
		if err != nil {
			return nil, fmt.Errorf("transform tx: %w", err)
		}

		txs = append(txs, t)
	}

	return txs, nil
}

func (b *Balances) PreviousTxs(
	ctx context.Context,
	balanceID uuid.UUID,
	includeDeleted bool,
	beforeUUID uuid.UUID,
	limit int,
) ([]domain.Tx, error) {
	rows, err := b.q.PreviousTxs(ctx, db.PreviousTxsParams{
		BalanceID:      balanceID,
		IncludeDeleted: includeDeleted,
		TxID:           beforeUUID,
		Limit:          int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("fetch txs: %w", err)
	}

	var txs []domain.Tx
	for _, r := range rows {
		t, err := transform.TxFromPgx(r)
		if err != nil {
			return nil, fmt.Errorf("transform tx: %w", err)
		}

		txs = append(txs, t)
	}

	return txs, nil
}

func (b *Balances) OpenBalance(ctx context.Context, balanceID uuid.UUID) error {
	if _, err := b.q.OpenBalance(ctx, balanceID); err != nil {
		if isPgCode(err, "23505") {
			return fmt.Errorf("%w: %v", ErrAlreadyExists, err)
		}
		return fmt.Errorf("open balance: %w", err)
	}

	return nil
}

func (b *Balances) Balance(ctx context.Context, balanceID uuid.UUID) (domain.Balance, error) {
	row, err := b.q.Balance(ctx, balanceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Balance{}, fmt.Errorf("%w: %v", ErrNotFound, err)
		}
		return domain.Balance{}, fmt.Errorf("fetch balance: %w", err)
	}

	balance, err := transform.BalanceFromPgx(row)
	if err != nil {
		return domain.Balance{}, fmt.Errorf("transform balance: %w", err)
	}

	return balance, nil
}

func isPgCode(err error, code string) bool {
	var pgerr *pgconn.PgError
	return errors.As(err, &pgerr) && pgerr.Code == code
}
