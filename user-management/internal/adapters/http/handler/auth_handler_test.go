package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/application"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/user"
)

type mockAuthService struct {
	registerErr   error
	registerCalls int
	registerName  string
	registerEmail string

	loginResult   *application.LoginResult
	loginErr      error
	loginCalls    int
	loginEmail    string
	loginPassword string
}

func (m *mockAuthService) Register(
	_ context.Context,
	name string,
	email string,
	password string,
) error {
	m.registerCalls++
	m.registerName = name
	m.registerEmail = email
	_ = password

	return m.registerErr
}

func (m *mockAuthService) Login(
	_ context.Context,
	email string,
	password string,
) (*application.LoginResult, error) {
	m.loginCalls++
	m.loginEmail = email
	m.loginPassword = password

	return m.loginResult, m.loginErr
}

func TestAuthHandler_Register(t *testing.T) {
	auth := &mockAuthService{}
	handler := NewAuthHandler(auth)

	response := performRequest(
		handler.Register,
		http.MethodPost,
		"/auth/register",
		`{"name":"Alice","email":"alice@example.com","password":"password"}`,
	)

	if response.Code != http.StatusCreated {
		t.Fatalf("Register() status = %d, want %d", response.Code, http.StatusCreated)
	}
	if auth.registerCalls != 1 {
		t.Errorf("Register() calls = %d, want 1", auth.registerCalls)
	}
	if auth.registerName != "Alice" {
		t.Errorf("Register() name = %q, want %q", auth.registerName, "Alice")
	}
	if auth.registerEmail != "alice@example.com" {
		t.Errorf("Register() email = %q, want %q", auth.registerEmail, "alice@example.com")
	}
}

func TestAuthHandler_Register_InvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "missing email", body: `{"name":"Alice","password":"password"}`},
		{name: "invalid email", body: `{"name":"Alice","email":"not-an-email","password":"password"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &mockAuthService{}
			handler := NewAuthHandler(auth)

			response := performRequest(
				handler.Register,
				http.MethodPost,
				"/auth/register",
				tt.body,
			)

			assertJSONError(t, response, http.StatusBadRequest, "invalid request")
			if auth.registerCalls != 0 {
				t.Errorf("Register() calls = %d, want 0", auth.registerCalls)
			}
		})
	}
}

func TestAuthHandler_Register_ServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		status     int
		message    string
	}{
		{
			name:       "invalid input",
			serviceErr: application.ErrInvalidInput,
			status:     http.StatusBadRequest,
			message:    "invalid input",
		},
		{
			name:       "email already exists",
			serviceErr: user.ErrEmailAlreadyExists,
			status:     http.StatusConflict,
			message:    "email already exists",
		},
		{
			name:       "internal error",
			serviceErr: errors.New("database unavailable"),
			status:     http.StatusInternalServerError,
			message:    "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &mockAuthService{registerErr: tt.serviceErr}
			handler := NewAuthHandler(auth)

			response := performRequest(
				handler.Register,
				http.MethodPost,
				"/auth/register",
				`{"name":"Alice","email":"alice@example.com","password":"password"}`,
			)

			assertJSONError(t, response, tt.status, tt.message)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	auth := &mockAuthService{
		loginResult: &application.LoginResult{
			Token: "jwt-token",
			User: application.LoginUser{
				ID:    "user-1",
				Name:  "Alice",
				Email: "alice@example.com",
			},
		},
	}
	handler := NewAuthHandler(auth)

	response := performRequest(
		handler.Login,
		http.MethodPost,
		"/auth/login",
		`{"email":"alice@example.com","password":"password"}`,
	)

	if response.Code != http.StatusOK {
		t.Fatalf("Login() status = %d, want %d", response.Code, http.StatusOK)
	}

	var body struct {
		Token string `json:"token"`
		User  struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"user"`
	}
	decodeJSON(t, response, &body)

	if body.Token != "jwt-token" {
		t.Errorf("response token = %q, want %q", body.Token, "jwt-token")
	}
	if body.User.ID != "user-1" {
		t.Errorf("response user ID = %q, want %q", body.User.ID, "user-1")
	}
	if body.User.Name != "Alice" {
		t.Errorf("response user name = %q, want %q", body.User.Name, "Alice")
	}
	if body.User.Email != "alice@example.com" {
		t.Errorf("response user email = %q, want %q", body.User.Email, "alice@example.com")
	}
	if auth.loginCalls != 1 {
		t.Errorf("Login() calls = %d, want 1", auth.loginCalls)
	}
}

func TestAuthHandler_Login_InvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "missing password", body: `{"email":"alice@example.com"}`},
		{name: "invalid email", body: `{"email":"not-an-email","password":"password"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &mockAuthService{}
			handler := NewAuthHandler(auth)

			response := performRequest(
				handler.Login,
				http.MethodPost,
				"/auth/login",
				tt.body,
			)

			assertJSONError(t, response, http.StatusBadRequest, "invalid request")
			if auth.loginCalls != 0 {
				t.Errorf("Login() calls = %d, want 0", auth.loginCalls)
			}
		})
	}
}

func TestAuthHandler_Login_ServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		status     int
		message    string
	}{
		{
			name:       "invalid credentials",
			serviceErr: application.ErrInvalidCredentials,
			status:     http.StatusUnauthorized,
			message:    "invalid email or password",
		},
		{
			name:       "internal error",
			serviceErr: errors.New("database unavailable"),
			status:     http.StatusInternalServerError,
			message:    "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &mockAuthService{loginErr: tt.serviceErr}
			handler := NewAuthHandler(auth)

			response := performRequest(
				handler.Login,
				http.MethodPost,
				"/auth/login",
				`{"email":"alice@example.com","password":"password"}`,
			)

			assertJSONError(t, response, tt.status, tt.message)
		})
	}
}

func performRequest(
	handler gin.HandlerFunc,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, path, handler)

	recording := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recording, request)

	return recording
}

func assertJSONError(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantStatus int,
	wantMessage string,
) {
	t.Helper()

	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d", response.Code, wantStatus)
	}

	var body struct {
		Error string `json:"error"`
	}
	decodeJSON(t, response, &body)

	if body.Error != wantMessage {
		t.Errorf("error message = %q, want %q", body.Error, wantMessage)
	}
}

func decodeJSON(t *testing.T, response *httptest.ResponseRecorder, destination any) {
	t.Helper()

	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode response JSON: %v", err)
	}
}
