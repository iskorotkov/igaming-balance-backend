package service

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	balancev1 "github.com/iskorotkov/igaming-balance-backend/gen/balance/v1"
	"github.com/iskorotkov/igaming-balance-backend/internal/domain"
	"github.com/iskorotkov/igaming-balance-backend/internal/storage"
	"github.com/iskorotkov/igaming-balance-backend/internal/transform"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Storage interface {
	RecordTx(ctx context.Context, tx domain.Tx) error
	CancelTxs(ctx context.Context, balanceID uuid.UUID, txIDs []uuid.UUID) error
	RecentTxs(ctx context.Context, balanceID uuid.UUID, includeDeleted bool, limit int) ([]domain.Tx, error)
	PreviousTxs(ctx context.Context, balanceID uuid.UUID, includeDeleted bool, before uuid.UUID, limit int) ([]domain.Tx, error)
	OpenBalance(ctx context.Context, balanceID uuid.UUID) error
	Balance(ctx context.Context, balanceID uuid.UUID) (domain.Balance, error)
}

func NewBalances(s Storage) *Balances {
	return &Balances{
		s: s,
	}
}

type Balances struct {
	s Storage
}

func (b *Balances) ListTx(
	ctx context.Context,
	req *connect.Request[balancev1.ListTxRequest],
) (*connect.Response[balancev1.ListTxResponse], error) {
	balanceID, err := uuid.Parse(req.Msg.GetBalanceId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var txs []domain.Tx
	if req.Msg.GetPageToken() == "" {
		txs, err = b.s.RecentTxs(ctx, balanceID, req.Msg.GetIncludeDeleted(), int(req.Msg.PageSize))
		if err != nil {
			slog.Error("failed to get recent transactions", "error", err)
			return nil, connect.NewError(connect.CodeInternal, errors.New("failed to get transactions"))
		}
	} else {
		beforeUUID, err := uuid.Parse(req.Msg.GetPageToken())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		txs, err = b.s.PreviousTxs(ctx, balanceID, req.Msg.GetIncludeDeleted(), beforeUUID, int(req.Msg.PageSize))
		if err != nil {
			slog.Error("failed to get previous transactions", "error", err)
			return nil, connect.NewError(connect.CodeInternal, errors.New("failed to get transactions"))
		}
	}

	if len(txs) == 0 {
		return connect.NewResponse(&balancev1.ListTxResponse{
			Txs:           nil,
			NextPageToken: "",
		}), nil
	}

	protoTxs := make([]*balancev1.Tx, 0, len(txs))
	for _, tx := range txs {
		t, err := transform.TxToProto(tx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		protoTxs = append(protoTxs, t)
	}

	return connect.NewResponse(&balancev1.ListTxResponse{
		Txs:           protoTxs,
		NextPageToken: protoTxs[len(protoTxs)-1].TxId,
	}), nil
}

func (b *Balances) RecordTx(
	ctx context.Context,
	req *connect.Request[balancev1.RecordTxRequest],
) (*connect.Response[emptypb.Empty], error) {
	tx, err := transform.TxFromProto(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := b.s.RecordTx(ctx, tx); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("balance not found"))
		}
		if errors.Is(err, storage.ErrAlreadyExists) {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("transaction already exists"))
		}
		if errors.Is(err, storage.ErrNegativeBalance) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("negative balance"))
		}
		slog.Error("failed to record transaction", "error", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to record transaction"))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (b *Balances) CancelTxs(
	ctx context.Context,
	req *connect.Request[balancev1.CancelTxsRequest],
) (*connect.Response[emptypb.Empty], error) {
	if len(req.Msg.GetTxIds()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no transaction ids provided"))
	}

	balanceID, err := uuid.Parse(req.Msg.GetBalanceId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	txIDs := make([]uuid.UUID, 0, len(req.Msg.GetTxIds()))
	for _, txID := range req.Msg.GetTxIds() {
		id, err := uuid.Parse(txID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		txIDs = append(txIDs, id)
	}

	if err := b.s.CancelTxs(ctx, balanceID, txIDs); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("transactions not found"))
		}
		if errors.Is(err, storage.ErrNegativeBalance) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("negative balance"))
		}
		slog.Error("failed to cancel transactions", "error", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to cancel transactions"))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (b *Balances) OpenBalance(
	ctx context.Context,
	req *connect.Request[balancev1.OpenBalanceRequest],
) (*connect.Response[emptypb.Empty], error) {
	balanceID, err := uuid.Parse(req.Msg.GetBalanceId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := b.s.OpenBalance(ctx, balanceID); err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("balance already open"))
		}
		slog.Error("failed to open balance", "error", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to open balance"))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (b *Balances) Balance(
	ctx context.Context,
	req *connect.Request[balancev1.BalanceRequest],
) (*connect.Response[balancev1.BalanceResponse], error) {
	balanceID, err := uuid.Parse(req.Msg.GetBalanceId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	balance, err := b.s.Balance(ctx, balanceID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("balance not found"))
		}
		slog.Error("failed to get balance", "error", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to get balance"))
	}

	protoBalance, err := transform.BalanceToProto(balance)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(protoBalance), nil
}
