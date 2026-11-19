package server

import (
	"context"
	"fmt"

	"example.com/config"
	"github.com/gofiber/fiber/v3"
)

func RunServer(ctx context.Context, app *fiber.App, cfg config.ServerConfig) error {
	listenErr := make(chan error, 1)
	go func() {
		listenErr <- app.Listen(cfg.Addr)
	}()

	select {
	case err := <-listenErr:
		return fmt.Errorf("listen on %s: %w", cfg.Addr, err)
	case <-ctx.Done():
	}

	if err := app.ShutdownWithTimeout(cfg.ShutdownTimeout); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	return <-listenErr
}
