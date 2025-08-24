package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"github.com/caarlos0/env/v11"
	"github.com/iskorotkov/igaming-balance-backend/gen/balance/v1/balancev1connect"
	"github.com/iskorotkov/igaming-balance-backend/internal/db"
	"github.com/iskorotkov/igaming-balance-backend/internal/middleware"
	"github.com/iskorotkov/igaming-balance-backend/internal/service"
	"github.com/iskorotkov/igaming-balance-backend/internal/storage"
	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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
}

func run(ctx context.Context, c Config) error {
	reflector := grpcreflect.NewStaticReflector(
		balancev1connect.BalanceServiceName,
	)

	pgxConfig, err := pgxpool.ParseConfig(c.DB)
	if err != nil {
		return fmt.Errorf("parse database config: %w", err)
	}
	pgxConfig.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		pgxdecimal.Register(c.TypeMap())
		return nil
	}

	conn, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer conn.Close()

	queries := db.New(conn)
	storage := storage.NewBalances(conn, queries)
	service := service.NewBalances(storage)

	mux := http.NewServeMux()
	mux.Handle(balancev1connect.NewBalanceServiceHandler(service,
		connect.WithInterceptors(middleware.LogRequests()),
	))
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true)

	server := &http.Server{
		Addr:         c.Addr,
		Handler:      h2c.NewHandler(mux, &http2.Server{}),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Protocols:    &protocols,
	}

	slog.InfoContext(ctx, "starting server", "addr", c.Addr)
	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		slog.InfoContext(ctx, "stopping server")
		if err := server.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "failed to shutdown server", "error", err)
		}
		slog.InfoContext(ctx, "server stopped")
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}
