package auth

import (
	"context"

	"example.com/internal/users"
)

type userLookup interface {
	FindOneByEmail(ctx context.Context, email string) (*users.User, error)
}
