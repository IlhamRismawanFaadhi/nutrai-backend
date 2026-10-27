package handler

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"

	"nutrai/internal/dto"
	"nutrai/internal/middleware"
	"nutrai/internal/n8n"
	"nutrai/internal/response"
	"nutrai/internal/service"
)

type AIHandler struct {
	svc       *service.AIService
	maxUpload int64
}

func NewAIHandler(s *service.AIService, maxUpload int64) *AIHandler {
	return &AIHandler{svc: s, maxUpload: maxUpload}
}

var allowedImages = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

// POST /ai/scan  (multipart/form-data, field "image")
func (h *AIHandler) Scan(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUpload+(1<<20))
	fh, err := c.FormFile("image")
	if err != nil {
		response.Fail(c, 400, "invalid_upload", "field multipart 'image' wajib diisi (maks ukuran file terlampaui juga memicu error ini)")
		return
	}
	if fh.Size > h.maxUpload {
		response.Fail(c, 413, "file_too_large", "ukuran foto melebihi batas")
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Fail(c, 400, "invalid_upload", "gagal membaca file")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, h.maxUpload+1))
	if err != nil || int64(len(data)) > h.maxUpload {
		response.Fail(c, 400, "invalid_upload", "gagal membaca file")
		return
	}
	ctype := http.DetectContentType(data) // jangan percaya header dari client
	if !allowedImages[ctype] {
		response.Fail(c, 415, "unsupported_media", "format harus JPEG, PNG, atau WebP")
		return
	}

	res, err := h.svc.Scan(c.Request.Context(), middleware.CurrentUser(c), fh.Filename, ctype, data)
	if err != nil {
		writeAIError(c, err)
		return
	}
	response.OK(c, 200, res)
}

// POST /ai/chat
func (h *AIHandler) Chat(c *gin.Context) {
	var req dto.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "validation_error", err.Error())
		return
	}
	res, err := h.svc.Chat(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		writeAIError(c, err)
		return
	}
	response.OK(c, 200, res)
}

func writeAIError(c *gin.Context, err error) {
	var n8nErr *n8n.Error
	var netErr net.Error
	switch {
	case errors.Is(err, service.ErrAINotConfigured):
		response.Fail(c, 503, "ai_not_configured", "layanan AI belum dikonfigurasi")
	case errors.Is(err, service.ErrNoFoodDetected):
		response.Fail(c, 422, "no_food_detected", "makanan tidak terdeteksi, coba foto lain atau isi manual")
	case errors.Is(err, context.Canceled):
		c.AbortWithStatus(499) // client menutup koneksi
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		response.Fail(c, 504, "ai_timeout", "layanan AI terlalu lama merespons")
	case errors.As(err, &n8nErr), errors.Is(err, service.ErrAIBadResponse):
		response.Fail(c, 502, "ai_upstream_error", "layanan AI mengembalikan respons yang tidak valid")
	default:
		response.Fail(c, 500, "internal_error", "terjadi kesalahan")
	}
	_ = c.Error(err) // tercatat di log Gin
}
