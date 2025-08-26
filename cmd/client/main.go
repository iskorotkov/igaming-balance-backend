package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/caarlos0/env/v11"
	"github.com/google/uuid"
	balancev1 "github.com/iskorotkov/igaming-balance-backend/gen/balance/v1"
	"github.com/iskorotkov/igaming-balance-backend/gen/balance/v1/balancev1connect"
	"github.com/iskorotkov/igaming-balance-backend/internal/middleware"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "panic: %v\n", err)
			os.Exit(1)
		}
	}()

	config, err := env.ParseAs[Config]()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: config.LogLevel,
	})))

	if err := run(ctx, config); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type Config struct {
	LogLevel slog.Level `env:"LOG_LEVEL"`
	Addr     string     `env:"ADDR"`
	DB       string     `env:"DB"`

	CreateInterval time.Duration `env:"CREATE_INTERVAL"`
	CreateCount    int           `env:"CREATE_COUNT"`
	CreateAmount   float64       `env:"CREATE_AMOUNT"`

	CancelInterval time.Duration `env:"CANCEL_INTERVAL"`
	CancelCount    int           `env:"CANCEL_COUNT"`
}

func run(ctx context.Context, c Config) error {
	client := balancev1connect.NewBalanceServiceClient(&http.Client{}, c.Addr,
		connect.WithGRPC(),
		connect.WithInterceptors(middleware.LogRequests()),
	)

	balanceID, err := uuid.NewV7() // UUID v7 are automatically sorted by timestamp.
	if err != nil {
		return fmt.Errorf("generate balance ID: %w", err)
	}

	if _, err := client.OpenBalance(ctx, &connect.Request[balancev1.OpenBalanceRequest]{
		Msg: &balancev1.OpenBalanceRequest{
			BalanceId: balanceID.String(),
		},
	}); err != nil {
		return fmt.Errorf("open balance: %w", err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		createTxs(ctx, c, client, balanceID)
	})
	wg.Go(func() {
		cancelTxs(ctx, c, client, balanceID)
	})

	wg.Wait()

	return nil
}

func createTxs(
	ctx context.Context,
	c Config,
	client balancev1connect.BalanceServiceClient,
	balanceID uuid.UUID,
) {
	ticker := time.NewTicker(c.CreateInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			slog.InfoContext(ctx, "creating transactions")

			for i := range c.CreateCount {
				txID, err := uuid.NewV7()
				if err != nil {
					slog.ErrorContext(ctx, "generate transaction ID", "error", err, "i", i)
					continue
				}

				tx := &balancev1.RecordTxRequest{
					BalanceId: balanceID.String(),
					TxId:      txID.String(),
					Source:    balancev1.Source(1 + rand.IntN(3)),
					State:     balancev1.State(1 + rand.IntN(2)),
					Amount: &balancev1.Decimal{
						Value: strconv.FormatFloat(rand.NormFloat64()*c.CreateAmount, 'f', -1, 64),
					},
				}

				if _, err := client.RecordTx(ctx, &connect.Request[balancev1.RecordTxRequest]{
					Msg: tx,
				}); err != nil {
					slog.ErrorContext(ctx, "create transaction", "error", err, "i", i, "tx", tx.String())
					continue
				}
			}

			slog.InfoContext(ctx, "created transactions", "count", c.CreateCount)
		}
	}
}

func cancelTxs(
	ctx context.Context,
	c Config,
	client balancev1connect.BalanceServiceClient,
	balanceID uuid.UUID,
) {
	ticker := time.NewTicker(c.CancelInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			slog.InfoContext(ctx, "canceling transactions")

			lastTxs, err := client.ListTx(ctx, &connect.Request[balancev1.ListTxRequest]{
				Msg: &balancev1.ListTxRequest{
					BalanceId: balanceID.String(),
					PageSize:  int32(c.CancelCount * 2),
				},
			})
			if err != nil {
				slog.ErrorContext(ctx, "list transactions", "error", err)
				continue
			}

			txsToCancel := make([]string, 0, len(lastTxs.Msg.GetTxs())/2)
			for i, tx := range lastTxs.Msg.GetTxs() {
				if i%2 == 1 {
					txsToCancel = append(txsToCancel, tx.GetTxId())
				}
			}

			if _, err := client.CancelTxs(ctx, &connect.Request[balancev1.CancelTxsRequest]{
				Msg: &balancev1.CancelTxsRequest{
					BalanceId: balanceID.String(),
					TxIds:     txsToCancel,
				},
			}); err != nil {
				slog.ErrorContext(ctx, "cancel transactions", "error", err)
				continue
			}

			slog.InfoContext(ctx, "cancelled transactions", "count", len(txsToCancel))
		}
	}
}
