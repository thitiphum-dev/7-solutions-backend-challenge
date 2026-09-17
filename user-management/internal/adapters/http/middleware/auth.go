package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const UserIDKey = "user_id"

type TokenVerifier interface {
	Verify(token string) (userID string, err error)
}

type Auth struct {
	tokens TokenVerifier
}

func NewAuth(tokens TokenVerifier) *Auth {
	return &Auth{
		tokens: tokens,
	}
}

func (m *Auth) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")

		scheme, token, ok := strings.Cut(header, " ")
		if !ok ||
			!strings.EqualFold(scheme, "Bearer") ||
			token == "" {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "unauthorized"},
			)
			return
		}

		userID, err := m.tokens.Verify(token)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{"error": "unauthorized"},
			)
			return
		}

		c.Set(UserIDKey, userID)

		c.Next()
	}
}
