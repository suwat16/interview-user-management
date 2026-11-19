package users

import "context"

type Service interface {
	Find(ctx context.Context) ([]User, error)
	FindOneByID(ctx context.Context, id string) (*User, error)
	FindOneByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, input CreateInput) (*User, error)
	Update(ctx context.Context, id string, fields UpdateFields) (*User, error)
	Delete(ctx context.Context, user *User) (*User, error)
}
