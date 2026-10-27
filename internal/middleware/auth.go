package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"nutrai/internal/models"
	"nutrai/internal/pkg/apikey"
	"nutrai/internal/repository"
	"nutrai/internal/response"
)

const userCtxKey = "auth_user"

// APIKeyAuth menerima `X-API-KEY: <key>` atau `Authorization: Bearer <key>`.
func APIKeyAuth(repo *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader("X-API-KEY"))
		if key == "" {
			if a := c.GetHeader("Authorization"); len(a) > 7 && strings.EqualFold(a[:7], "bearer ") {
				key = strings.TrimSpace(a[7:])
			}
		}
		if key == "" {
			response.Fail(c, 401, "unauthorized", "API key wajib disertakan (X-API-KEY atau Authorization: Bearer)")
			c.Abort()
			return
		}
		user, err := repo.FindUserByKeyHash(apikey.Hash(key))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.Fail(c, 401, "unauthorized", "API key tidak valid")
			} else {
				response.Fail(c, 500, "internal_error", "gagal memverifikasi API key")
			}
			c.Abort()
			return
		}
		c.Set(userCtxKey, user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *models.User {
	v, _ := c.Get(userCtxKey)
	u, _ := v.(*models.User)
	return u
}
