package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/http/middleware"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/application"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/user"
)

type mockUserService struct {
	listResult  []application.UserResult
	listErr     error
	getResult   *application.UserResult
	getErr      error
	updateErr   error
	deleteErr   error
	getCalls    int
	updateCalls int
	deleteCalls int
	updateID    string
	updateName  *string
	updateEmail *string
}

func (m *mockUserService) List(_ context.Context) ([]application.UserResult, error) {
	return m.listResult, m.listErr
}

func (m *mockUserService) GetByID(
	_ context.Context,
	_ string,
) (*application.UserResult, error) {
	m.getCalls++
	return m.getResult, m.getErr
}

func (m *mockUserService) UpdateByID(
	_ context.Context,
	id string,
	name *string,
	email *string,
) error {
	m.updateCalls++
	m.updateID = id
	m.updateName = name
	m.updateEmail = email
	return m.updateErr
}

func (m *mockUserService) DeleteByID(
	_ context.Context,
	_ string,
) error {
	m.deleteCalls++
	return m.deleteErr
}

func TestUserHandler_List(t *testing.T) {
	createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	service := &mockUserService{
		listResult: []application.UserResult{
			{
				ID:        "user-1",
				Name:      "Alice",
				Email:     "alice@example.com",
				CreatedAt: createdAt,
			},
		},
	}
	handler := NewUserHandler(service)

	response := performRequest(handler.List, http.MethodGet, "/users", "")

	if response.Code != http.StatusOK {
		t.Fatalf("List() status = %d, want %d", response.Code, http.StatusOK)
	}

	var body []userResponse
	decodeJSON(t, response, &body)
	if len(body) != 1 {
		t.Fatalf("List() response length = %d, want 1", len(body))
	}
	if body[0].ID != "user-1" || body[0].CreatedAt != createdAt {
		t.Errorf("List() response = %+v", body[0])
	}
}

func TestUserHandler_List_Error(t *testing.T) {
	service := &mockUserService{listErr: errors.New("database unavailable")}
	handler := NewUserHandler(service)

	response := performRequest(handler.List, http.MethodGet, "/users", "")
	assertJSONError(t, response, http.StatusInternalServerError, "internal server error")
}

func TestUserHandler_GetByID(t *testing.T) {
	service := &mockUserService{
		getResult: &application.UserResult{
			ID:    "user-1",
			Name:  "Alice",
			Email: "alice@example.com",
		},
	}
	handler := NewUserHandler(service)

	response := performUserRouteRequest(
		handler.GetByID,
		http.MethodGet,
		"/users/:id",
		"/users/user-1",
		"",
	)

	if response.Code != http.StatusOK {
		t.Fatalf("GetByID() status = %d, want %d", response.Code, http.StatusOK)
	}

	var body userResponse
	decodeJSON(t, response, &body)
	if body.ID != "user-1" || body.Name != "Alice" {
		t.Errorf("GetByID() response = %+v", body)
	}
}

func TestUserHandler_GetByID_Errors(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		status     int
		message    string
	}{
		{
			name:       "not found",
			serviceErr: user.ErrNotFound,
			status:     http.StatusNotFound,
			message:    "user not found",
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
			handler := NewUserHandler(&mockUserService{getErr: tt.serviceErr})
			response := performUserRouteRequest(
				handler.GetByID,
				http.MethodGet,
				"/users/:id",
				"/users/user-1",
				"",
			)
			assertJSONError(t, response, tt.status, tt.message)
		})
	}
}

func TestUserHandler_UpdateByID(t *testing.T) {
	service := &mockUserService{}
	handler := NewUserHandler(service)

	response := performUserRouteRequest(
		handler.UpdateByID,
		http.MethodPatch,
		"/users/:id",
		"/users/user-1",
		`{"name":"Alice Updated","email":"alice.updated@example.com"}`,
	)

	if response.Code != http.StatusNoContent {
		t.Fatalf("UpdateByID() status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if service.updateCalls != 1 {
		t.Fatalf("UpdateByID() calls = %d, want 1", service.updateCalls)
	}
	if service.updateID != "user-1" {
		t.Errorf("UpdateByID() ID = %q, want %q", service.updateID, "user-1")
	}
	if service.updateName == nil || *service.updateName != "Alice Updated" {
		t.Errorf("UpdateByID() name = %v", service.updateName)
	}
	if service.updateEmail == nil || *service.updateEmail != "alice.updated@example.com" {
		t.Errorf("UpdateByID() email = %v", service.updateEmail)
	}
}

func TestUserHandler_UpdateByID_InvalidRequest(t *testing.T) {
	service := &mockUserService{}
	handler := NewUserHandler(service)

	response := performUserRouteRequest(
		handler.UpdateByID,
		http.MethodPatch,
		"/users/:id",
		"/users/user-1",
		`{"email":"not-an-email"}`,
	)

	assertJSONError(t, response, http.StatusBadRequest, "invalid request")
	if service.updateCalls != 0 {
		t.Errorf("UpdateByID() calls = %d, want 0", service.updateCalls)
	}
}

func TestUserHandler_UpdateByID_Forbidden(t *testing.T) {
	service := &mockUserService{}
	handler := NewUserHandler(service)

	response := performUserRouteRequestAs(
		handler.UpdateByID,
		http.MethodPatch,
		"/users/:id",
		"/users/user-1",
		"user-2",
		`{"name":"Alice Updated"}`,
	)

	assertJSONError(t, response, http.StatusForbidden, "forbidden")
	if service.updateCalls != 0 {
		t.Errorf("UpdateByID() calls = %d, want 0", service.updateCalls)
	}
}

func TestUserHandler_UpdateByID_Errors(t *testing.T) {
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
			name:       "not found",
			serviceErr: user.ErrNotFound,
			status:     http.StatusNotFound,
			message:    "user not found",
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
			handler := NewUserHandler(&mockUserService{updateErr: tt.serviceErr})
			response := performUserRouteRequest(
				handler.UpdateByID,
				http.MethodPatch,
				"/users/:id",
				"/users/user-1",
				`{"name":"Alice Updated"}`,
			)
			assertJSONError(t, response, tt.status, tt.message)
		})
	}
}

func TestUserHandler_DeleteByID(t *testing.T) {
	service := &mockUserService{}
	handler := NewUserHandler(service)

	response := performUserRouteRequest(
		handler.DeleteByID,
		http.MethodDelete,
		"/users/:id",
		"/users/user-1",
		"",
	)

	if response.Code != http.StatusNoContent {
		t.Fatalf("DeleteByID() status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if service.deleteCalls != 1 {
		t.Errorf("DeleteByID() calls = %d, want 1", service.deleteCalls)
	}
}

func TestUserHandler_DeleteByID_Forbidden(t *testing.T) {
	service := &mockUserService{}
	handler := NewUserHandler(service)

	response := performUserRouteRequestAs(
		handler.DeleteByID,
		http.MethodDelete,
		"/users/:id",
		"/users/user-1",
		"user-2",
		"",
	)

	assertJSONError(t, response, http.StatusForbidden, "forbidden")
	if service.deleteCalls != 0 {
		t.Errorf("DeleteByID() calls = %d, want 0", service.deleteCalls)
	}
}

func TestUserHandler_DeleteByID_Errors(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		status     int
		message    string
	}{
		{
			name:       "not found",
			serviceErr: user.ErrNotFound,
			status:     http.StatusNotFound,
			message:    "user not found",
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
			handler := NewUserHandler(&mockUserService{deleteErr: tt.serviceErr})
			response := performUserRouteRequest(
				handler.DeleteByID,
				http.MethodDelete,
				"/users/:id",
				"/users/user-1",
				"",
			)
			assertJSONError(t, response, tt.status, tt.message)
		})
	}
}

func performUserRouteRequest(
	handlerFunc gin.HandlerFunc,
	method string,
	route string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	return performUserRouteRequestAs(
		handlerFunc,
		method,
		route,
		path,
		"user-1",
		body,
	)
}

func performUserRouteRequestAs(
	handlerFunc gin.HandlerFunc,
	method string,
	route string,
	path string,
	authenticatedUserID string,
	body string,
) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.UserIDKey, authenticatedUserID)
		c.Next()
	})
	router.Handle(method, route, handlerFunc)

	recording := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recording, request)

	return recording
}
