package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/user"
)

type UserService struct {
	users user.Repository
}

func NewUserService(users user.Repository) *UserService {
	return &UserService{
		users: users,
	}
}

type UserResult struct {
	ID        string
	Name      string
	Email     string
	CreatedAt time.Time
}

func (s *UserService) List(ctx context.Context) ([]UserResult, error) {
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	result := make([]UserResult, 0, len(users))

	for _, u := range users {
		result = append(result, toUserResult(u))
	}

	return result, nil
}

func (s *UserService) GetByID(
	ctx context.Context,
	id string,
) (*UserResult, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	result := toUserResult(u)

	return &result, nil
}

func (s *UserService) UpdateByID(
	ctx context.Context,
	id string,
	name *string,
	email *string,
) error {
	if name == nil && email == nil {
		return ErrInvalidInput
	}

	update := user.Update{}

	if name != nil {
		normalized := strings.TrimSpace(*name)
		if normalized == "" {
			return ErrInvalidInput
		}

		update.Name = &normalized
	}

	if email != nil {
		normalized := strings.ToLower(strings.TrimSpace(*email))
		if normalized == "" {
			return ErrInvalidInput
		}

		update.Email = &normalized
	}

	if err := s.users.UpdateByID(ctx, id, update); err != nil {
		return fmt.Errorf("update user by id: %w", err)
	}

	return nil
}

func (s *UserService) DeleteByID(
	ctx context.Context,
	id string,
) error {
	if err := s.users.DeleteByID(ctx, id); err != nil {
		return fmt.Errorf("delete user by id: %w", err)
	}

	return nil
}

func toUserResult(u *user.User) UserResult {
	return UserResult{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}
