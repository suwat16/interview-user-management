package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/internal/users"
	"example.com/internal/users/handler"
	"example.com/internal/users/service"
	"example.com/pkg/middleware"
	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

const userID = "test-user-id"
const email = "ada@example.com"
const password = "test-password"

type repositoryMock struct {
	users.Repository
	save   func(*users.User) error
	find   func() ([]users.User, error)
	byID   func(string) (*users.User, error)
	update func(string, users.UpdateFields) (*users.User, error)
	delete func(*users.User) error
}

func (r repositoryMock) Save(_ context.Context, user *users.User) error {
	return r.save(user)
}

func (r repositoryMock) Find(context.Context) ([]users.User, error) {
	return r.find()
}

func (r repositoryMock) FindByID(_ context.Context, id string) (*users.User, error) {
	return r.byID(id)
}

func (r repositoryMock) Update(_ context.Context, id string, fields users.UpdateFields) (*users.User, error) {
	return r.update(id, fields)
}

func (r repositoryMock) Delete(_ context.Context, user *users.User) error {
	return r.delete(user)
}

func TestCreateUser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    string
		saveErr error
		status  int
	}{
		{"created", `{"name":" Ada ","email":" ada@example.com ","password":"test-password"}`, nil, http.StatusCreated},
		{"duplicate email", `{"name":"Ada","email":"ada@example.com","password":"test-password"}`, users.ErrEmailTaken, http.StatusConflict},
		{"invalid email", `{"name":"Ada","email":"invalid","password":"test-password"}`, nil, http.StatusBadRequest},
		{"missing password", `{"name":"Ada","email":"ada@example.com"}`, nil, http.StatusBadRequest},
		{"malformed JSON", `{"name":`, nil, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			saveCalled := false
			repo := repositoryMock{save: func(user *users.User) error {
				saveCalled = true
				if user.Name != "Ada" || user.Email != email {
					t.Errorf("stored name and email = %q, %q", user.Name, user.Email)
				}
				if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
					t.Errorf("stored password is not a matching bcrypt hash: %v", err)
				}
				if _, err := time.Parse(time.RFC3339, user.CreatedAt); err != nil {
					t.Errorf("created_at is not an RFC3339 timestamp: %v", err)
				}
				user.ID = userID
				return tc.saveErr
			}}
			app := newUsersApp(repo, []byte("unit-test-key"))
			res := request(t, app, http.MethodPost, "/users", tc.body, "", tc.status)
			if saveCalled != (tc.status != http.StatusBadRequest) {
				t.Errorf("repository Save called = %v", saveCalled)
			}
			if tc.status == http.StatusCreated {
				assertUser(t, res, "Ada", email)
			}
		})
	}
}

func TestUserRoutes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{"list users", http.MethodGet, "/users", "", http.StatusOK},
		{"empty list", http.MethodGet, "/users", "", http.StatusOK},
		{"get user", http.MethodGet, "/users/" + userID, "", http.StatusOK},
		{"get missing user", http.MethodGet, "/users/missing", "", http.StatusNotFound},
		{"update name", http.MethodPatch, "/users/" + userID, `{"name":" Grace "}`, http.StatusOK},
		{"update email", http.MethodPatch, "/users/" + userID, `{"email":" grace@example.com "}`, http.StatusOK},
		{"update to duplicate email", http.MethodPatch, "/users/" + userID, `{"email":"other@example.com"}`, http.StatusConflict},
		{"empty update", http.MethodPatch, "/users/" + userID, `{}`, http.StatusBadRequest},
		{"update missing user", http.MethodPatch, "/users/missing", `{"name":"Grace"}`, http.StatusNotFound},
		{"delete user", http.MethodDelete, "/users/" + userID, "", http.StatusNoContent},
		{"delete missing user", http.MethodDelete, "/users/missing", "", http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			user := users.User{ID: userID, Name: "Ada", Email: email, Password: "stored-hash"}
			calls := 0
			repo := repositoryMock{
				find: func() ([]users.User, error) {
					calls++
					if tc.name == "empty list" {
						return nil, nil
					}
					return []users.User{user}, nil
				},
				byID: func(id string) (*users.User, error) {
					calls++
					if id != userID {
						return nil, users.ErrUserNotFound
					}
					return &user, nil
				},
				update: func(id string, fields users.UpdateFields) (*users.User, error) {
					calls++
					if id != userID {
						return nil, users.ErrUserNotFound
					}
					switch tc.name {
					case "update to duplicate email":
						if fields.Email == nil || *fields.Email != "other@example.com" || fields.Name != nil {
							t.Error("expected only the submitted email to be updated")
						}
						return nil, users.ErrEmailTaken
					case "update email":
						if fields.Email == nil || *fields.Email != "grace@example.com" || fields.Name != nil {
							t.Error("expected only the trimmed email to be updated")
						}
						user.Email = "grace@example.com"
					default:
						if fields.Name == nil || *fields.Name != "Grace" || fields.Email != nil {
							t.Error("expected only the trimmed name to be updated")
						}
						user.Name = "Grace"
					}
					return &user, nil
				},
				delete: func(user *users.User) error {
					calls++
					if user.ID != userID {
						return users.ErrUserNotFound
					}
					return nil
				},
			}
			key := []byte("unit-test-key")
			app := newUsersApp(repo, key)
			request(t, app, tc.method, tc.path, tc.body, "", http.StatusUnauthorized)
			if calls != 0 {
				t.Fatal("unauthenticated request reached the repository")
			}
			token, err := middleware.IssueAccessToken(userID, key, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			res := request(t, app, tc.method, tc.path, tc.body, token, tc.status)
			wantCalls := 1
			if tc.status == http.StatusBadRequest {
				wantCalls = 0
			}
			if calls != wantCalls {
				t.Errorf("repository calls = %d, want %d", calls, wantCalls)
			}
			switch tc.name {
			case "list users", "empty list":
				var list []userResponse
				decode(t, res, &list)
				if tc.name == "empty list" {
					if list == nil || len(list) != 0 {
						t.Fatalf("expected an empty JSON array, got %#v", list)
					}
				} else if len(list) != 1 || list[0].ID != userID || list[0].Password != nil {
					t.Fatalf("unexpected user list: %#v", list)
				}
			case "get user":
				assertUser(t, res, "Ada", email)
			case "update name":
				assertUser(t, res, "Grace", email)
			case "update email":
				assertUser(t, res, "Ada", "grace@example.com")
			}
		})
	}
}

type userResponse struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Email    string          `json:"email"`
	Password json.RawMessage `json:"password"`
}

func assertUser(t *testing.T, res *http.Response, name, email string) {
	t.Helper()
	var user userResponse
	decode(t, res, &user)
	if user.ID != userID || user.Name != name || user.Email != email || user.Password != nil {
		t.Fatalf("unexpected user response: %#v", user)
	}
}

func decode(t *testing.T, res *http.Response, dst any) {
	t.Helper()
	if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
		t.Fatal(err)
	}
}

func newUsersApp(repo users.Repository, key []byte) *fiber.App {
	app := fiber.New()
	handler.NewHandler(service.NewUserService(repo), middleware.RequireJWT(key)).RegisterRoute(app)
	return app
}

func request(t *testing.T, app *fiber.App, method, path, body, token string, status int) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// Bcrypt can take longer than Fiber's one-second default under the race detector.
	res, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second, FailOnTimeout: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := res.Body.Close(); err != nil {
			t.Error(err)
		}
	})
	if res.StatusCode != status {
		t.Fatalf("%s %s status = %d, want %d", method, path, res.StatusCode, status)
	}
	return res
}
