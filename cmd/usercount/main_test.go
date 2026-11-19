package usercount_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"example.com/cmd/usercount"
	"example.com/config"
)

func TestLogUserCount(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		cfg := config.UserCountConfig{Interval: 10 * time.Second, Timeout: time.Second}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		started := time.Now()
		calls := 0
		done := make(chan struct{})

		go func() {
			defer close(done)
			usercount.LogUserCount(ctx, cfg, func(queryCtx context.Context) (int64, error) {
				calls++
				deadline, ok := queryCtx.Deadline()
				if !ok || time.Until(deadline) != cfg.Timeout {
					t.Errorf("query deadline = %v, want %v from now", deadline, cfg.Timeout)
				}
				if elapsed := time.Since(started); elapsed != time.Duration(calls)*cfg.Interval {
					t.Errorf("count %d ran after %v", calls, elapsed)
				}
				if calls == 1 {
					return 0, errors.New("count failed")
				}
				return 5, nil
			})
		}()

		time.Sleep(2 * cfg.Interval)
		synctest.Wait()
		if calls != 2 {
			t.Errorf("count calls = %d, want 2 even when the first query fails", calls)
		}
		cancel()
		<-done
	})
}
