package response

import (
	"log"

	"github.com/gofiber/fiber/v3"
)

const internalErrorMessage = "internal server error"

type ErrorBody struct {
	Error string `json:"error"`
}

func OK(c fiber.Ctx, data any) error {
	return c.Status(fiber.StatusOK).JSON(data)
}

func Created(c fiber.Ctx, data any) error {
	return c.Status(fiber.StatusCreated).JSON(data)
}

func NoContent(c fiber.Ctx) error {
	return c.Status(fiber.StatusNoContent).Send(nil)
}

func Error(c fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(ErrorBody{Error: msg})
}

func InternalError(c fiber.Ctx, err error) error {
	log.Printf("%s %s: %v", c.Method(), c.Path(), err)
	return Error(c, fiber.StatusInternalServerError, internalErrorMessage)
}
