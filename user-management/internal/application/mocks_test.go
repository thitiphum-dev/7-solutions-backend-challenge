package application

import (
	"context"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/user"
)

type mockUserRepository struct {
	createdUser       *user.User
	createErr         error
	findByEmailResult *user.User
	findByEmailErr    error
	findByEmailCalls  int
}

func (m *mockUserRepository) Create(
	_ context.Context,
	u *user.User,
) error {
	m.createdUser = u
	return m.createErr
}

func (m *mockUserRepository) FindByEmail(
	_ context.Context,
	_ string,
) (*user.User, error) {
	m.findByEmailCalls++
	return m.findByEmailResult, m.findByEmailErr
}

func (m *mockUserRepository) FindByID(
	_ context.Context,
	_ string,
) (*user.User, error) {
	return nil, user.ErrNotFound
}

func (m *mockUserRepository) List(
	_ context.Context,
) ([]*user.User, error) {
	return nil, nil
}

func (m *mockUserRepository) UpdateByID(
	_ context.Context,
	_ string,
	_ user.Update,
) error {
	return nil
}

func (m *mockUserRepository) DeleteByID(
	_ context.Context,
	_ string,
) error {
	return nil
}

func (m *mockUserRepository) Count(
	_ context.Context,
) (int64, error) {
	return 0, nil
}

type mockPasswordHasher struct {
	hashInput       string
	hashResult      string
	hashErr         error
	hashCalls       int
	compareHash     string
	comparePassword string
	compareErr      error
	compareCalls    int
}

func (m *mockPasswordHasher) Hash(password string) (string, error) {
	m.hashCalls++
	m.hashInput = password
	return m.hashResult, m.hashErr
}

func (m *mockPasswordHasher) Compare(hash, password string) error {
	m.compareCalls++
	m.compareHash = hash
	m.comparePassword = password
	return m.compareErr
}

type mockTokenIssuer struct {
	token      string
	err        error
	userID     string
	issueCalls int
}

func (m *mockTokenIssuer) Issue(userID string) (string, error) {
	m.issueCalls++
	m.userID = userID
	return m.token, m.err
}
