package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"example.com/config"
	"example.com/internal/users"
	"example.com/pkg/middleware"
	"example.com/pkg/response"
	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	users           userLookup
	secret          []byte
	missingUserHash []byte
	tokenLifetime   time.Duration
}

func NewHandler(lookup userLookup, cfg config.AuthConfig) (*Handler, error) {
	if len(cfg.JWTSecret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 bytes")
	}
	missingUserHash := []byte(cfg.MissingUserHash)
	if cost, err := bcrypt.Cost(missingUserHash); err != nil || cost != bcrypt.DefaultCost {
		return nil, fmt.Errorf("MISSING_USER_HASH must be a bcrypt hash with cost %d", bcrypt.DefaultCost)
	}
	return &Handler{
		users:           lookup,
		secret:          []byte(cfg.JWTSecret),
		missingUserHash: missingUserHash,
		tokenLifetime:   cfg.AccessTokenLifetime,
	}, nil
}

func (h *Handler) RegisterRoute(router fiber.Router) {
	router.Post("/auth/login", h.login)
}

func (h *Handler) login(c fiber.Ctx) error {
	var request loginRequest
	if err := c.Bind().JSON(&request); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid JSON body")
	}

	email := strings.TrimSpace(request.Email)
	if !validLoginInput(email, request.Password) {
		return response.Error(c, fiber.StatusBadRequest, "valid email and password are required")
	}

	user, err := h.users.FindOneByEmail(c.Context(), email)
	if err != nil && !errors.Is(err, users.ErrUserNotFound) {
		if errors.Is(err, users.ErrEmailTaken) {
			return response.Error(c, fiber.StatusConflict, users.ErrEmailTaken.Error())
		}
		return response.InternalError(c, err)
	}

	hash := h.missingUserHash
	if err == nil {
		hash = []byte(user.Password)
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(request.Password)) != nil || err != nil {
		return response.Error(c, fiber.StatusUnauthorized, users.ErrInvalidCredentials.Error())
	}

	token, err := middleware.IssueAccessToken(user.ID, h.secret, h.tokenLifetime)
	if err != nil {
		return response.InternalError(c, err)
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return response.OK(c, loginResponse{AccessToken: token})
}
