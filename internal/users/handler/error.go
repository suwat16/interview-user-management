package handler

import (
	"errors"

	"example.com/internal/users"
	"example.com/pkg/response"
	"github.com/gofiber/fiber/v3"
)

func invalidJSONError(c fiber.Ctx) error {
	return response.Error(c, fiber.StatusBadRequest, "invalid JSON body")
}

func userError(c fiber.Ctx, err error) error {
	var invalidInput users.ValidationError
	switch {
	case errors.As(err, &invalidInput):
		return response.Error(c, fiber.StatusBadRequest, invalidInput.Error())
	case errors.Is(err, users.ErrUserNotFound):
		return response.Error(c, fiber.StatusNotFound, users.ErrUserNotFound.Error())
	case errors.Is(err, users.ErrEmailTaken):
		return response.Error(c, fiber.StatusConflict, users.ErrEmailTaken.Error())
	default:
		return response.InternalError(c, err)
	}
}
