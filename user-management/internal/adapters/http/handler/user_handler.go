package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/http/middleware"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/application"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/user"
)

const (
	internalServerErrorMessage = "internal server error"
	forbiddenMessage           = "forbidden"
	userNotFoundMessage        = "user not found"
)

type UserService interface {
	List(ctx context.Context) ([]application.UserResult, error)
	GetByID(ctx context.Context, id string) (*application.UserResult, error)
	UpdateByID(
		ctx context.Context,
		id string,
		name *string,
		email *string,
	) error
	DeleteByID(ctx context.Context, id string) error
}

type UserHandler struct {
	users UserService
}

func NewUserHandler(users UserService) *UserHandler {
	return &UserHandler{
		users: users,
	}
}

type userResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *UserHandler) List(c *gin.Context) {
	users, err := h.users.List(c.Request.Context())
	if err != nil {
		slog.Error("failed to list users", "error", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": internalServerErrorMessage,
		})
		return
	}

	response := make([]userResponse, 0, len(users))

	for _, u := range users {
		response = append(response, toUserResponse(u))
	}

	c.JSON(http.StatusOK, response)
}

func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	u, err := h.users.GetByID(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": userNotFoundMessage,
			})

		default:
			slog.Error(
				"failed to get user",
				"user_id", id,
				"error", err,
			)

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": internalServerErrorMessage,
			})
		}

		return
	}

	c.JSON(http.StatusOK, toUserResponse(*u))
}

type updateUserRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email" binding:"omitempty,email"`
}

func (h *UserHandler) UpdateByID(c *gin.Context) {
	if !ensureUserOwnsResource(c) {
		return
	}

	id := c.Param("id")

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	err := h.users.UpdateByID(
		c.Request.Context(),
		id,
		req.Name,
		req.Email,
	)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid input",
			})

		case errors.Is(err, user.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": userNotFoundMessage,
			})

		case errors.Is(err, user.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": "email already exists",
			})

		default:
			slog.Error(
				"failed to update user",
				"user_id", id,
				"error", err,
			)

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": internalServerErrorMessage,
			})
		}

		return
	}

	c.Status(http.StatusNoContent)
}

func (h *UserHandler) DeleteByID(c *gin.Context) {
	if !ensureUserOwnsResource(c) {
		return
	}

	id := c.Param("id")

	if err := h.users.DeleteByID(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": userNotFoundMessage,
			})

		default:
			slog.Error(
				"failed to delete user",
				"user_id", id,
				"error", err,
			)

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": internalServerErrorMessage,
			})
		}

		return
	}

	c.Status(http.StatusNoContent)
}

func toUserResponse(u application.UserResult) userResponse {
	return userResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

func ensureUserOwnsResource(c *gin.Context) bool {
	id := c.Param("id")
	authenticatedUserID := c.GetString(middleware.UserIDKey)

	if authenticatedUserID == id {
		return true
	}

	c.JSON(http.StatusForbidden, gin.H{
		"error": forbiddenMessage,
	})
	return false
}
