package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/application"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/user"
)

type AuthService interface {
	Register(
		ctx context.Context,
		name string,
		email string,
		password string,
	) error

	Login(
		ctx context.Context,
		email string,
		password string,
	) (*application.LoginResult, error)
}

type AuthHandler struct {
	auth AuthService
}

func NewAuthHandler(auth AuthService) *AuthHandler {
	return &AuthHandler{
		auth: auth,
	}
}

type registerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	if err := h.auth.Register(
		c.Request.Context(),
		req.Name,
		req.Email,
		req.Password,
	); err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid input",
			})

		case errors.Is(err, user.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": "email already exists",
			})

		default:
			slog.Error("failed to register user", "error", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}

		return
	}

	c.Status(http.StatusCreated)
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	Token string            `json:"token"`
	User  loginUserResponse `json:"user"`
}

type loginUserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	result, err := h.auth.Login(
		c.Request.Context(),
		req.Email,
		req.Password,
	)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid email or password",
			})

		default:
			slog.Error("failed to login user", "error", err)

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
		}

		return
	}

	c.JSON(http.StatusOK, loginResponse{
		Token: result.Token,
		User: loginUserResponse{
			ID:    result.User.ID,
			Name:  result.User.Name,
			Email: result.User.Email,
		},
	})
}
