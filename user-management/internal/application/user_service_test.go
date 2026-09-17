package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/user"
)

type userServiceRepositoryMock struct {
	listResult  []*user.User
	listErr     error
	findResult  *user.User
	findErr     error
	updateErr   error
	deleteErr   error
	updateID    string
	update      user.Update
	deleteID    string
	updateCalls int
	deleteCalls int
}

func (m *userServiceRepositoryMock) Create(
	_ context.Context,
	_ *user.User,
) error {
	return nil
}

func (m *userServiceRepositoryMock) FindByEmail(
	_ context.Context,
	_ string,
) (*user.User, error) {
	return nil, user.ErrNotFound
}

func (m *userServiceRepositoryMock) FindByID(
	_ context.Context,
	_ string,
) (*user.User, error) {
	return m.findResult, m.findErr
}

func (m *userServiceRepositoryMock) List(
	_ context.Context,
) ([]*user.User, error) {
	return m.listResult, m.listErr
}

func (m *userServiceRepositoryMock) UpdateByID(
	_ context.Context,
	id string,
	update user.Update,
) error {
	m.updateCalls++
	m.updateID = id
	m.update = update
	return m.updateErr
}

func (m *userServiceRepositoryMock) DeleteByID(
	_ context.Context,
	id string,
) error {
	m.deleteCalls++
	m.deleteID = id
	return m.deleteErr
}

func (m *userServiceRepositoryMock) Count(
	_ context.Context,
) (int64, error) {
	return 0, nil
}

func TestUserService_List(t *testing.T) {
	createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	repository := &userServiceRepositoryMock{
		listResult: []*user.User{
			{
				ID:           "user-1",
				Name:         "Alice",
				Email:        "alice@example.com",
				PasswordHash: "should-not-be-exposed",
				CreatedAt:    createdAt,
			},
		},
	}
	service := NewUserService(repository)

	result, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("List() result length = %d, want 1", len(result))
	}
	if result[0].ID != "user-1" {
		t.Errorf("List() ID = %q, want %q", result[0].ID, "user-1")
	}
	if result[0].CreatedAt != createdAt {
		t.Errorf("List() CreatedAt = %v, want %v", result[0].CreatedAt, createdAt)
	}
}

func TestUserService_List_Error(t *testing.T) {
	listErr := errors.New("database unavailable")
	service := NewUserService(&userServiceRepositoryMock{listErr: listErr})

	_, err := service.List(context.Background())
	if !errors.Is(err, listErr) {
		t.Fatalf("List() error = %v, want wrapped repository error", err)
	}
}

func TestUserService_GetByID(t *testing.T) {
	repository := &userServiceRepositoryMock{
		findResult: &user.User{
			ID:    "user-1",
			Name:  "Alice",
			Email: "alice@example.com",
		},
	}
	service := NewUserService(repository)

	result, err := service.GetByID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if result.ID != "user-1" || result.Name != "Alice" {
		t.Errorf("GetByID() result = %+v", result)
	}
}

func TestUserService_GetByID_Error(t *testing.T) {
	service := NewUserService(&userServiceRepositoryMock{
		findErr: user.ErrNotFound,
	})

	_, err := service.GetByID(context.Background(), "missing")
	if !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want %v", err, user.ErrNotFound)
	}
}

func TestUserService_UpdateByID(t *testing.T) {
	name := "  Alice Updated  "
	email := " ALICE.UPDATED@Example.COM "
	repository := &userServiceRepositoryMock{}
	service := NewUserService(repository)

	err := service.UpdateByID(
		context.Background(),
		"user-1",
		&name,
		&email,
	)
	if err != nil {
		t.Fatalf("UpdateByID() error = %v", err)
	}

	if repository.updateCalls != 1 {
		t.Fatalf("UpdateByID() calls = %d, want 1", repository.updateCalls)
	}
	if repository.updateID != "user-1" {
		t.Errorf("UpdateByID() ID = %q, want %q", repository.updateID, "user-1")
	}
	if repository.update.Name == nil || *repository.update.Name != "Alice Updated" {
		t.Errorf("UpdateByID() name = %v, want %q", repository.update.Name, "Alice Updated")
	}
	if repository.update.Email == nil || *repository.update.Email != "alice.updated@example.com" {
		t.Errorf("UpdateByID() email = %v, want %q", repository.update.Email, "alice.updated@example.com")
	}
}

func TestUserService_UpdateByID_InvalidInput(t *testing.T) {
	blank := "   "
	tests := []struct {
		name  string
		field string
	}{
		{name: "no fields", field: "none"},
		{name: "blank name", field: "name"},
		{name: "blank email", field: "email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &userServiceRepositoryMock{}
			service := NewUserService(repository)

			var name *string
			var email *string
			switch tt.field {
			case "name":
				name = &blank
			case "email":
				email = &blank
			}

			err := service.UpdateByID(
				context.Background(),
				"user-1",
				name,
				email,
			)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("UpdateByID() error = %v, want %v", err, ErrInvalidInput)
			}
			if repository.updateCalls != 0 {
				t.Errorf("UpdateByID() calls = %d, want 0", repository.updateCalls)
			}
		})
	}
}

func TestUserService_UpdateByID_Error(t *testing.T) {
	updateErr := user.ErrEmailAlreadyExists
	repository := &userServiceRepositoryMock{updateErr: updateErr}
	service := NewUserService(repository)
	name := "Alice Updated"

	err := service.UpdateByID(
		context.Background(),
		"user-1",
		&name,
		nil,
	)
	if !errors.Is(err, updateErr) {
		t.Fatalf("UpdateByID() error = %v, want wrapped repository error", err)
	}
}

func TestUserService_DeleteByID(t *testing.T) {
	repository := &userServiceRepositoryMock{}
	service := NewUserService(repository)

	err := service.DeleteByID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("DeleteByID() error = %v", err)
	}
	if repository.deleteCalls != 1 {
		t.Errorf("DeleteByID() calls = %d, want 1", repository.deleteCalls)
	}
	if repository.deleteID != "user-1" {
		t.Errorf("DeleteByID() ID = %q, want %q", repository.deleteID, "user-1")
	}
}

func TestUserService_DeleteByID_Error(t *testing.T) {
	deleteErr := user.ErrNotFound
	service := NewUserService(&userServiceRepositoryMock{deleteErr: deleteErr})

	err := service.DeleteByID(context.Background(), "missing")
	if !errors.Is(err, deleteErr) {
		t.Fatalf("DeleteByID() error = %v, want wrapped repository error", err)
	}
}
