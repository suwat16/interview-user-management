package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
)

func LogRequest(c fiber.Ctx) error {
	method, path := c.Method(), c.Path()
	start := time.Now()
	err := c.Next()
	log.Printf("method=%q path=%q duration=%s", method, path, time.Since(start))
	return err
}
