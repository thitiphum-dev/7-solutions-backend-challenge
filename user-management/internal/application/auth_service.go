package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/user"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenIssuer interface {
	Issue(userID string) (string, error)
}

type AuthService struct {
	users  user.Repository
	hasher PasswordHasher
	tokens TokenIssuer
}

func NewAuthService(
	users user.Repository,
	hasher PasswordHasher,
	tokens TokenIssuer,
) *AuthService {
	return &AuthService{
		users:  users,
		hasher: hasher,
		tokens: tokens,
	}
}

func (s *AuthService) Register(
	ctx context.Context,
	name string,
	email string,
	password string,
) error {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))

	if name == "" || email == "" || password == "" {
		return ErrInvalidInput
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	u := &user.User{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.users.Create(ctx, u); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

type LoginResult struct {
	Token string
	User  LoginUser
}

type LoginUser struct {
	ID    string
	Name  string
	Email string
}

func (s *AuthService) Login(
	ctx context.Context,
	email string,
	password string,
) (*LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("find user by email: %w", err)
	}

	if err := s.hasher.Compare(u.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(u.ID)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &LoginResult{
		Token: token,
		User: LoginUser{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		},
	}, nil
}
