package users

import (
	"context"
)

type Repository interface {
	Find(ctx context.Context) ([]User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Save(ctx context.Context, user *User) error
	Update(ctx context.Context, id string, fields UpdateFields) (*User, error)
	Delete(ctx context.Context, user *User) error
}
