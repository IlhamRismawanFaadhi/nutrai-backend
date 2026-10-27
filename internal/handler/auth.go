package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"nutrai/internal/dto"
	"nutrai/internal/middleware"
	"nutrai/internal/response"
	"nutrai/internal/service"
)

type AuthHandler struct{ svc *service.AuthService }

func NewAuthHandler(s *service.AuthService) *AuthHandler { return &AuthHandler{svc: s} }

// POST /auth/register (publik)
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "validation_error", err.Error())
		return
	}
	u, key, err := h.svc.Register(req.Name, req.Email)
	if errors.Is(err, service.ErrEmailTaken) {
		response.Fail(c, 409, "email_taken", "email sudah terdaftar")
		return
	}
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal registrasi")
		return
	}
	response.OK(c, 201, dto.AuthResponse{
		UserID: u.ID, Name: u.Name, Email: u.Email, APIKey: key,
		Note: "Simpan API key ini sekarang; tidak akan ditampilkan lagi.",
	})
}

// POST /auth/generate-key (butuh API key lama) => rotasi key.
func (h *AuthHandler) GenerateKey(c *gin.Context) {
	u := middleware.CurrentUser(c)
	key, err := h.svc.RotateKey(u.ID)
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal membuat API key")
		return
	}
	response.OK(c, 200, dto.AuthResponse{
		UserID: u.ID, Name: u.Name, Email: u.Email, APIKey: key,
		Note: "API key lama sudah tidak berlaku. Simpan key baru ini.",
	})
}
