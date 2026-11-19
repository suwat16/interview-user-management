package handler

import (
	"example.com/internal/users"
	"example.com/pkg/response"
	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service    users.Service
	requireJWT fiber.Handler
}

func NewHandler(service users.Service, requireJWT fiber.Handler) *Handler {
	return &Handler{service: service, requireJWT: requireJWT}
}

func (h *Handler) RegisterRoute(router fiber.Router) {
	group := router.Group("/users")
	group.Post("/", h.create)
	group.Use(h.requireJWT)
	group.Get("/", h.list)
	group.Get("/:id", h.findByID)
	group.Patch("/:id", h.update)
	group.Delete("/:id", h.delete)
}

func (h *Handler) create(c fiber.Ctx) error {
	var request RegisterRequest
	if err := c.Bind().JSON(&request); err != nil {
		return invalidJSONError(c)
	}

	user, err := h.service.Create(c.Context(), users.CreateInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		return userError(c, err)
	}
	return response.Created(c, user)
}

func (h *Handler) list(c fiber.Ctx) error {
	result, err := h.service.Find(c.Context())
	if err != nil {
		return userError(c, err)
	}
	if result == nil {
		result = []users.User{}
	}
	return response.OK(c, result)
}

func (h *Handler) findByID(c fiber.Ctx) error {
	user, err := h.service.FindOneByID(c.Context(), c.Params("id"))
	if err != nil {
		return userError(c, err)
	}
	return response.OK(c, user)
}

func (h *Handler) update(c fiber.Ctx) error {
	var request updateRequest
	if err := c.Bind().JSON(&request); err != nil {
		return invalidJSONError(c)
	}
	user, err := h.service.Update(c.Context(), c.Params("id"), users.UpdateFields{
		Name:  request.Name,
		Email: request.Email,
	})
	if err != nil {
		return userError(c, err)
	}
	return response.OK(c, user)
}

func (h *Handler) delete(c fiber.Ctx) error {
	if _, err := h.service.Delete(c.Context(), &users.User{ID: c.Params("id")}); err != nil {
		return userError(c, err)
	}
	return response.NoContent(c)
}
