package user

import "context"

type Repository interface {
	Create(ctx context.Context, user *User) error

	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context) ([]*User, error)

	UpdateByID(ctx context.Context, id string, update Update) error
	DeleteByID(ctx context.Context, id string) error

	Count(ctx context.Context) (int64, error)
}
