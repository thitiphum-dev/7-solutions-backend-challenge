package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/user"
)

func TestAuthService_Register(t *testing.T) {
	repository := &mockUserRepository{}
	hasher := &mockPasswordHasher{
		hashResult: "hashed-password",
	}
	service := NewAuthService(repository, hasher, &mockTokenIssuer{})

	err := service.Register(
		context.Background(),
		"  Alice  ",
		" ALICE@Example.COM ",
		"plain-password",
	)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if repository.createdUser == nil {
		t.Fatal("Register() did not create a user")
	}

	created := repository.createdUser
	if created.Name != "Alice" {
		t.Errorf("created user name = %q, want %q", created.Name, "Alice")
	}
	if created.Email != "alice@example.com" {
		t.Errorf("created user email = %q, want %q", created.Email, "alice@example.com")
	}
	if created.PasswordHash != "hashed-password" {
		t.Errorf("created user password hash = %q, want %q", created.PasswordHash, "hashed-password")
	}
	if created.CreatedAt.IsZero() {
		t.Error("created user CreatedAt is zero")
	}
	if hasher.hashInput != "plain-password" {
		t.Errorf("Hash() input = %q, want %q", hasher.hashInput, "plain-password")
	}
}

func TestAuthService_Register_InvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		password string
	}{
		{name: "empty name", email: "alice@example.com", password: "password"},
		{name: "empty email", username: "Alice", password: "password"},
		{name: "empty password", username: "Alice", email: "alice@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &mockUserRepository{}
			hasher := &mockPasswordHasher{hashResult: "hashed-password"}
			service := NewAuthService(repository, hasher, &mockTokenIssuer{})

			err := service.Register(
				context.Background(),
				tt.username,
				tt.email,
				tt.password,
			)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Register() error = %v, want %v", err, ErrInvalidInput)
			}
			if hasher.hashCalls != 0 {
				t.Errorf("Hash() calls = %d, want 0", hasher.hashCalls)
			}
			if repository.createdUser != nil {
				t.Error("Register() created a user for invalid input")
			}
		})
	}
}

func TestAuthService_Register_HashError(t *testing.T) {
	hashErr := errors.New("hash failed")
	repository := &mockUserRepository{}
	hasher := &mockPasswordHasher{hashErr: hashErr}
	service := NewAuthService(repository, hasher, &mockTokenIssuer{})

	err := service.Register(
		context.Background(),
		"Alice",
		"alice@example.com",
		"password",
	)
	if !errors.Is(err, hashErr) {
		t.Fatalf("Register() error = %v, want wrapped hash error", err)
	}
	if repository.createdUser != nil {
		t.Error("Register() created a user after hashing failed")
	}
}

func TestAuthService_Register_CreateError(t *testing.T) {
	createErr := user.ErrEmailAlreadyExists
	repository := &mockUserRepository{createErr: createErr}
	hasher := &mockPasswordHasher{hashResult: "hashed-password"}
	service := NewAuthService(repository, hasher, &mockTokenIssuer{})

	err := service.Register(
		context.Background(),
		"Alice",
		"alice@example.com",
		"password",
	)
	if !errors.Is(err, createErr) {
		t.Fatalf("Register() error = %v, want wrapped create error", err)
	}
}

func TestAuthService_Login(t *testing.T) {
	repository := &mockUserRepository{
		findByEmailResult: &user.User{
			ID:           "user-1",
			Name:         "Alice",
			Email:        "alice@example.com",
			PasswordHash: "stored-hash",
			CreatedAt:    time.Now().UTC(),
		},
	}
	hasher := &mockPasswordHasher{}
	tokens := &mockTokenIssuer{token: "jwt-token"}
	service := NewAuthService(repository, hasher, tokens)

	result, err := service.Login(
		context.Background(),
		" ALICE@Example.COM ",
		"plain-password",
	)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if result.Token != "jwt-token" {
		t.Errorf("Login() token = %q, want %q", result.Token, "jwt-token")
	}
	if result.User.ID != "user-1" {
		t.Errorf("Login() user ID = %q, want %q", result.User.ID, "user-1")
	}
	if result.User.Name != "Alice" {
		t.Errorf("Login() user name = %q, want %q", result.User.Name, "Alice")
	}
	if result.User.Email != "alice@example.com" {
		t.Errorf("Login() user email = %q, want %q", result.User.Email, "alice@example.com")
	}
	if hasher.compareHash != "stored-hash" {
		t.Errorf("Compare() hash = %q, want %q", hasher.compareHash, "stored-hash")
	}
	if hasher.comparePassword != "plain-password" {
		t.Errorf("Compare() password = %q, want %q", hasher.comparePassword, "plain-password")
	}
	if tokens.userID != "user-1" {
		t.Errorf("Issue() user ID = %q, want %q", tokens.userID, "user-1")
	}
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name          string
		findResult    *user.User
		findErr       error
		compareErr    error
		wantFindCalls int
	}{
		{
			name:          "user not found",
			findErr:       user.ErrNotFound,
			wantFindCalls: 1,
		},
		{
			name: "password mismatch",
			findResult: &user.User{
				ID:           "user-1",
				PasswordHash: "stored-hash",
			},
			compareErr:    errors.New("password mismatch"),
			wantFindCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &mockUserRepository{
				findByEmailResult: tt.findResult,
				findByEmailErr:    tt.findErr,
			}
			hasher := &mockPasswordHasher{compareErr: tt.compareErr}
			tokens := &mockTokenIssuer{token: "should-not-be-issued"}
			service := NewAuthService(repository, hasher, tokens)

			result, err := service.Login(
				context.Background(),
				"alice@example.com",
				"wrong-password",
			)
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want %v", err, ErrInvalidCredentials)
			}
			if result != nil {
				t.Error("Login() result is not nil for invalid credentials")
			}
			if repository.findByEmailCalls != tt.wantFindCalls {
				t.Errorf("FindByEmail() calls = %d, want %d", repository.findByEmailCalls, tt.wantFindCalls)
			}
			if tokens.issueCalls != 0 {
				t.Errorf("Issue() calls = %d, want 0", tokens.issueCalls)
			}
		})
	}
}

func TestAuthService_Login_FindError(t *testing.T) {
	findErr := errors.New("database unavailable")
	repository := &mockUserRepository{findByEmailErr: findErr}
	service := NewAuthService(
		repository,
		&mockPasswordHasher{},
		&mockTokenIssuer{},
	)

	_, err := service.Login(
		context.Background(),
		"alice@example.com",
		"password",
	)
	if !errors.Is(err, findErr) {
		t.Fatalf("Login() error = %v, want wrapped find error", err)
	}
}

func TestAuthService_Login_TokenError(t *testing.T) {
	tokenErr := errors.New("token issuer unavailable")
	repository := &mockUserRepository{
		findByEmailResult: &user.User{
			ID:           "user-1",
			PasswordHash: "stored-hash",
		},
	}
	service := NewAuthService(
		repository,
		&mockPasswordHasher{},
		&mockTokenIssuer{err: tokenErr},
	)

	_, err := service.Login(
		context.Background(),
		"alice@example.com",
		"password",
	)
	if !errors.Is(err, tokenErr) {
		t.Fatalf("Login() error = %v, want wrapped token error", err)
	}
}

func TestAuthService_Login_EmptyCredentials(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
	}{
		{name: "empty email", password: "password"},
		{name: "empty password", email: "alice@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &mockUserRepository{}
			service := NewAuthService(
				repository,
				&mockPasswordHasher{},
				&mockTokenIssuer{},
			)

			result, err := service.Login(
				context.Background(),
				tt.email,
				tt.password,
			)
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want %v", err, ErrInvalidCredentials)
			}
			if result != nil {
				t.Error("Login() result is not nil for empty credentials")
			}
			if repository.findByEmailCalls != 0 {
				t.Errorf("FindByEmail() calls = %d, want 0", repository.findByEmailCalls)
			}
		})
	}
}
