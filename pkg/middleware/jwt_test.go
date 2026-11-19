package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"example.com/pkg/middleware"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func TestRequireJWT(t *testing.T) {
	t.Parallel()
	key := []byte("unit-test-key")
	expiresAt := jwt.NewNumericDate(time.Now().Add(time.Hour))
	valid := signJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: "user-id", ExpiresAt: expiresAt}, key)
	expired := signJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: "user-id", ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}, key)
	wrongKey := signJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: "user-id", ExpiresAt: expiresAt}, []byte("other-test-key"))
	wrongAlgorithm := signJWT(t, jwt.SigningMethodHS384, jwt.RegisteredClaims{Subject: "user-id", ExpiresAt: expiresAt}, key)
	noExpiry := signJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: "user-id"}, key)
	noSubject := signJWT(t, jwt.SigningMethodHS256, jwt.RegisteredClaims{ExpiresAt: expiresAt}, key)

	tests := []struct {
		name          string
		authorization string
		status        int
	}{
		{"valid bearer token", "Bearer " + valid, http.StatusNoContent},
		{"missing header", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic " + valid, http.StatusUnauthorized},
		{"malformed token", "Bearer invalid-token", http.StatusUnauthorized},
		{"expired token", "Bearer " + expired, http.StatusUnauthorized},
		{"wrong signing key", "Bearer " + wrongKey, http.StatusUnauthorized},
		{"unexpected signing algorithm", "Bearer " + wrongAlgorithm, http.StatusUnauthorized},
		{"missing expiration", "Bearer " + noExpiry, http.StatusUnauthorized},
		{"missing subject", "Bearer " + noSubject, http.StatusUnauthorized},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			app := fiber.New()
			app.Get("/private", middleware.RequireJWT(key), func(c fiber.Ctx) error {
				return c.SendStatus(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/private", nil)
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := res.Body.Close(); err != nil {
					t.Error(err)
				}
			}()
			if res.StatusCode != tc.status {
				t.Fatalf("status = %d, want %d", res.StatusCode, tc.status)
			}
			if tc.status == http.StatusUnauthorized && res.Header.Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", res.Header.Get("WWW-Authenticate"))
			}
		})
	}
}

func signJWT(t *testing.T, method jwt.SigningMethod, claims jwt.RegisteredClaims, key []byte) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return token
}
