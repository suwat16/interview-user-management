package middleware

import (
	"bytes"
	"strings"
	"time"

	"example.com/pkg/response"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func IssueAccessToken(userID string, secret []byte, lifetime time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(lifetime)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func RequireJWT(secret []byte) fiber.Handler {
	key := bytes.Clone(secret)

	return func(c fiber.Ctx) error {
		scheme, rawToken, ok := strings.Cut(c.Get(fiber.HeaderAuthorization), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") {
			return bearerRequired(c)
		}

		claims := new(jwt.RegisteredClaims)
		_, err := jwt.ParseWithClaims(strings.TrimSpace(rawToken), claims, func(*jwt.Token) (any, error) {
			return key, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
		if err != nil || strings.TrimSpace(claims.Subject) == "" {
			return bearerRequired(c)
		}
		return c.Next()
	}
}

func bearerRequired(c fiber.Ctx) error {
	c.Set(fiber.HeaderWWWAuthenticate, "Bearer")
	return response.Error(c, fiber.StatusUnauthorized, fiber.ErrUnauthorized.Message)
}
