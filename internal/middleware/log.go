package middleware

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
)

func LogRequests() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			attrs := []any{
				"req", req.Any(),
				"method", req.HTTPMethod(),
				"header", req.Header(),
				"peer", req.Peer(),
			}

			slog.DebugContext(ctx, "got request", attrs...)

			resp, err := next(ctx, req)
			if err != nil {
				attrs = append(attrs, "err", err)
				slog.DebugContext(ctx, "request not processed", attrs...)
				return resp, err
			}

			attrs = append(attrs, "resp", resp.Any())
			slog.DebugContext(ctx, "request processed", attrs...)
			return resp, nil
		}
	}
}
