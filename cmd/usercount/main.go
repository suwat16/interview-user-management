package usercount

import (
	"context"
	"log"
	"time"

	"example.com/config"
)

func LogUserCount(ctx context.Context, cfg config.UserCountConfig, countUsers func(context.Context) (int64, error)) {
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if ctx.Err() != nil {
			return
		}

		queryCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
		count, err := countUsers(queryCtx)
		cancel()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("user count failed: %v", err)
			continue
		}
		log.Printf("total_users=%d", count)
	}
}
