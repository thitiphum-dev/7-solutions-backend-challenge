package worker

import (
	"context"
	"log/slog"
	"time"
)

type UserCounter interface {
	Count(ctx context.Context) (int64, error)
}

func RunUserCount(
	ctx context.Context,
	counter UserCounter,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("user count worker stopped")
			return

		case <-ticker.C:
			countCtx, cancel := context.WithTimeout(ctx, 3*time.Second)

			count, err := counter.Count(countCtx)
			cancel()

			if err != nil {
				slog.Error(
					"failed to count users",
					"error", err,
				)
				continue
			}

			slog.Info(
				"user count",
				"count", count,
			)
		}
	}
}
