package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/internal/users"
	"example.com/internal/users/service"
	"golang.org/x/crypto/bcrypt"
)

type repositoryMock struct {
	find    func() ([]users.User, error)
	byID    func(string) (*users.User, error)
	byEmail func(string) (*users.User, error)
	save    func(*users.User) error
	update  func(string, users.UpdateFields) (*users.User, error)
	delete  func(*users.User) error
}

func (r repositoryMock) Find(context.Context) ([]users.User, error) {
	return r.find()
}

func (r repositoryMock) FindByID(_ context.Context, id string) (*users.User, error) {
	return r.byID(id)
}

func (r repositoryMock) FindByEmail(_ context.Context, email string) (*users.User, error) {
	return r.byEmail(email)
}

func (r repositoryMock) Save(_ context.Context, user *users.User) error {
	return r.save(user)
}

func (r repositoryMock) Update(_ context.Context, id string, fields users.UpdateFields) (*users.User, error) {
	return r.update(id, fields)
}

func (r repositoryMock) Delete(_ context.Context, user *users.User) error {
	return r.delete(user)
}

func TestCreate(t *testing.T) {
	t.Parallel()
	const password = " test-password "
	tests := []struct {
		name          string
		password      string
		saveErr       error
		validationErr bool
	}{
		{name: "saves a valid user", password: password},
		{name: "rejects empty password", validationErr: true},
		{name: "rejects password over bcrypt byte limit", password: strings.Repeat("é", 37), validationErr: true},
		{name: "returns duplicate email error", password: password, saveErr: users.ErrEmailTaken},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var saved *users.User
			repo := repositoryMock{save: func(user *users.User) error {
				saved = user
				user.ID = "assigned-id"
				return tc.saveErr
			}}
			got, err := service.NewUserService(repo).Create(t.Context(), users.CreateInput{
				Name: " Ada ", Email: " ada@example.com ", Password: tc.password,
			})
			if tc.validationErr {
				var validationErr users.ValidationError
				if !errors.As(err, &validationErr) || got != nil || saved != nil {
					t.Fatalf("expected validation error before Save, got user=%v, err=%v, Save called=%v", got != nil, err, saved != nil)
				}
				return
			}
			if saved == nil || saved.Name != "Ada" || saved.Email != "ada@example.com" {
				t.Fatal("expected Save to receive the user with trimmed name and email")
			}
			if !errors.Is(err, tc.saveErr) {
				t.Fatalf("Create error = %v, want %v", err, tc.saveErr)
			}
			if tc.saveErr != nil {
				if got != nil {
					t.Fatal("Create returned a user when Save failed")
				}
				return
			}
			if got == nil || *got != *saved || got.ID != "assigned-id" {
				t.Fatal("Create did not return the persisted user with its assigned ID")
			}
			if err := bcrypt.CompareHashAndPassword([]byte(saved.Password), []byte(tc.password)); err != nil {
				t.Fatalf("stored hash does not match the original password: %v", err)
			}
			if _, err := time.Parse(time.RFC3339, saved.CreatedAt); err != nil {
				t.Fatalf("invalid creation timestamp: %v", err)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		input         *string
		updateErr     error
		validationErr bool
	}{
		{name: "updates a trimmed name", input: new(" Grace ")},
		{name: "rejects empty update", validationErr: true},
		{name: "rejects blank name", input: new(" "), validationErr: true},
		{name: "returns missing user error", input: new(" Grace "), updateErr: users.ErrUserNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			want := users.User{ID: "user-id", Name: "Grace"}
			called := false
			repo := repositoryMock{update: func(id string, fields users.UpdateFields) (*users.User, error) {
				called = true
				if id != want.ID || fields.Name == nil || *fields.Name != want.Name || fields.Email != nil {
					t.Fatal("Update did not forward the ID and trimmed name")
				}
				if tc.updateErr != nil {
					return nil, tc.updateErr
				}
				return &want, nil
			}}
			got, err := service.NewUserService(repo).Update(t.Context(), want.ID, users.UpdateFields{Name: tc.input})
			if tc.validationErr {
				var validationErr users.ValidationError
				if !errors.As(err, &validationErr) || got != nil || called {
					t.Fatalf("expected validation error before Update, got user=%v, err=%v, Update called=%v", got, err, called)
				}
				return
			}
			if !called || !errors.Is(err, tc.updateErr) {
				t.Fatalf("Update called = %v, error = %v, want %v", called, err, tc.updateErr)
			}
			if tc.updateErr == nil && (got == nil || *got != want) {
				t.Fatalf("Update returned %v, want %+v", got, want)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"deletes a user", nil},
		{"returns missing user error", users.ErrUserNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			user := &users.User{ID: "user-id"}
			called := false
			repo := repositoryMock{delete: func(got *users.User) error {
				called = true
				if got == nil || got.ID != user.ID {
					t.Fatalf("Delete received %v, want user ID %q", got, user.ID)
				}
				return tc.err
			}}
			got, err := service.NewUserService(repo).Delete(t.Context(), user)
			if !called || !errors.Is(err, tc.err) {
				t.Fatalf("Delete called = %v, error = %v, want %v", called, err, tc.err)
			}
			if (tc.err == nil && (got == nil || *got != *user)) || (tc.err != nil && got != nil) {
				t.Fatalf("Delete returned unexpected user %v", got)
			}
		})
	}
}

func TestFind(t *testing.T) {
	t.Parallel()
	want := users.User{ID: "user-id"}
	repo := repositoryMock{find: func() ([]users.User, error) {
		return []users.User{want}, nil
	}}
	got, err := service.NewUserService(repo).Find(t.Context())
	if err != nil || len(got) != 1 || got[0] != want {
		t.Fatalf("Find = %v, %v, want one stored user", got, err)
	}
}

func TestFindOne(t *testing.T) {
	t.Parallel()
	want := users.User{ID: "user-id", Email: "ada@example.com"}
	tests := []struct {
		name  string
		find  func(users.Service, context.Context, string) (*users.User, error)
		query string
	}{
		{"by ID", users.Service.FindOneByID, want.ID},
		{"by email", users.Service.FindOneByEmail, want.Email},
	}
	repo := repositoryMock{
		byID: func(id string) (*users.User, error) {
			if id != want.ID {
				return nil, users.ErrUserNotFound
			}
			return &want, nil
		},
		byEmail: func(email string) (*users.User, error) {
			if email != want.Email {
				return nil, users.ErrUserNotFound
			}
			return &want, nil
		},
	}
	svc := service.NewUserService(repo)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.find(svc, t.Context(), tc.query)
			if err != nil || got == nil || *got != want {
				t.Fatalf("lookup = %v, %v, want stored user", got, err)
			}
			got, err = tc.find(svc, t.Context(), "missing")
			if got != nil || !errors.Is(err, users.ErrUserNotFound) {
				t.Fatalf("missing lookup = %v, %v, want ErrUserNotFound", got, err)
			}
		})
	}
}
