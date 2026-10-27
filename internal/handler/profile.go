package handler

import (
	"github.com/gin-gonic/gin"

	"nutrai/internal/dto"
	"nutrai/internal/middleware"
	"nutrai/internal/response"
	"nutrai/internal/service"
)

type ProfileHandler struct{ svc *service.ProfileService }

func NewProfileHandler(s *service.ProfileService) *ProfileHandler { return &ProfileHandler{svc: s} }

func (h *ProfileHandler) Get(c *gin.Context) {
	p, err := h.svc.Get(middleware.CurrentUser(c))
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal mengambil profil")
		return
	}
	response.OK(c, 200, p)
}

func (h *ProfileHandler) Update(c *gin.Context) {
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "validation_error", err.Error())
		return
	}
	p, err := h.svc.Update(middleware.CurrentUser(c), req)
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal menyimpan profil")
		return
	}
	response.OK(c, 200, p)
}
