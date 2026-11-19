package middleware

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
)

func RegisterHealthChecks(app *fiber.App, pingTimeout time.Duration, pingDB func(context.Context) error) {
	app.Get(healthcheck.LivenessEndpoint, healthcheck.New())
	app.Get(healthcheck.ReadinessEndpoint, healthcheck.New(healthcheck.Config{
		Probe: func(c fiber.Ctx) bool {
			ctx, cancel := context.WithTimeout(c.Context(), pingTimeout)
			defer cancel()
			return pingDB(ctx) == nil
		},
	}))
}
