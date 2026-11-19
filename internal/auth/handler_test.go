package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/config"
	"example.com/internal/auth"
	"example.com/internal/users"
	"example.com/pkg/response"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type lookupMock func(context.Context, string) (*users.User, error)

func (lookup lookupMock) FindOneByEmail(ctx context.Context, email string) (*users.User, error) {
	return lookup(ctx, email)
}

func TestLogin(t *testing.T) {
	t.Parallel()
	const userID = "test-user-id"
	const email = "ada@example.com"
	const validBody = `{"email":"ada@example.com","password":"test-password"}`
	hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.AuthConfig{
		JWTSecret:           strings.Repeat("a", 32),
		MissingUserHash:     string(hash),
		AccessTokenLifetime: time.Hour,
	}
	tests := []struct {
		name      string
		body      string
		lookupErr error
		status    int
		message   string
	}{
		{"valid credentials", `{"email":" ada@example.com ","password":"test-password"}`, nil, http.StatusOK, ""},
		{"wrong password", `{"email":"ada@example.com","password":"wrong-password"}`, nil, http.StatusUnauthorized, users.ErrInvalidCredentials.Error()},
		{"unknown email", validBody, users.ErrUserNotFound, http.StatusUnauthorized, users.ErrInvalidCredentials.Error()},
		{"missing password", `{"email":"ada@example.com"}`, nil, http.StatusBadRequest, "valid email and password are required"},
		{"invalid JSON", `{"email":`, nil, http.StatusBadRequest, "invalid JSON body"},
		{"repository failure", validBody, errors.New("database unavailable"), http.StatusInternalServerError, "internal server error"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			lookup := lookupMock(func(_ context.Context, address string) (*users.User, error) {
				calls++
				if address != email {
					t.Errorf("lookup email = %q, want %q", address, email)
				}
				if tc.lookupErr != nil {
					return nil, tc.lookupErr
				}
				return &users.User{ID: userID, Email: email, Password: string(hash)}, nil
			})
			handler, err := auth.NewHandler(lookup, cfg)
			if err != nil {
				t.Fatal(err)
			}
			app := fiber.New()
			handler.RegisterRoute(app)

			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second, FailOnTimeout: true})
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
			wantCalls := 1
			if tc.status == http.StatusBadRequest {
				wantCalls = 0
			}
			if calls != wantCalls {
				t.Errorf("user lookup calls = %d, want %d", calls, wantCalls)
			}
			if tc.status == http.StatusOK {
				assertLoginToken(t, res, cfg, userID)
				return
			}
			var body response.ErrorBody
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Error != tc.message {
				t.Errorf("error message = %q, want %q", body.Error, tc.message)
			}
		})
	}
}

func assertLoginToken(t *testing.T, res *http.Response, cfg config.AuthConfig, userID string) {
	t.Helper()
	if cache := res.Header.Get("Cache-Control"); cache != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cache)
	}
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	claims := new(jwt.RegisteredClaims)
	_, err := jwt.ParseWithClaims(body.AccessToken, claims, func(*jwt.Token) (any, error) {
		return []byte(cfg.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		t.Fatalf("invalid access token: %v", err)
	}
	if claims.Subject != userID || claims.IssuedAt == nil {
		t.Fatalf("unexpected access token claims: %+v", claims)
	}
	if lifetime := claims.ExpiresAt.Sub(claims.IssuedAt.Time); lifetime != cfg.AccessTokenLifetime {
		t.Errorf("token lifetime = %v, want %v", lifetime, cfg.AccessTokenLifetime)
	}
}
