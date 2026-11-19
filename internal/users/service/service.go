package service

import (
	"context"
	"fmt"
	"time"

	"example.com/internal/users"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo users.Repository
}

func NewUserService(repo users.Repository) users.Service {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) Find(ctx context.Context) ([]users.User, error) {
	return s.repo.Find(ctx)
}

func (s *UserService) FindOneByID(ctx context.Context, id string) (*users.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) FindOneByEmail(ctx context.Context, email string) (*users.User, error) {
	return s.repo.FindByEmail(ctx, email)
}

func (s *UserService) Create(ctx context.Context, input users.CreateInput) (*users.User, error) {
	input, err := input.Validate()
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash user password: %w", err)
	}
	user := &users.User{
		Name:      input.Name,
		Email:     input.Email,
		Password:  string(hash),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Update(ctx context.Context, id string, fields users.UpdateFields) (*users.User, error) {
	fields, err := fields.Validate()
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, fields)
}

func (s *UserService) Delete(ctx context.Context, user *users.User) (*users.User, error) {
	if err := s.repo.Delete(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
