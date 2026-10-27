package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"nutrai/internal/dto"
	"nutrai/internal/middleware"
	"nutrai/internal/response"
	"nutrai/internal/service"
)

type LogHandler struct{ svc *service.LogService }

func NewLogHandler(s *service.LogService) *LogHandler { return &LogHandler{svc: s} }

// GET /logs/today?date=YYYY-MM-DD (date opsional, untuk tombol Prev/Next di UI)
func (h *LogHandler) Today(c *gin.Context) {
	day, err := h.svc.ParseDate(c.Query("date"))
	if err != nil {
		response.Fail(c, 400, "validation_error", "format date harus YYYY-MM-DD")
		return
	}
	res, err := h.svc.Today(middleware.CurrentUser(c).ID, day)
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal mengambil log hari ini")
		return
	}
	response.OK(c, 200, res)
}

// POST /logs/meal
func (h *LogHandler) AddMeal(c *gin.Context) {
	var req dto.AddMealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "validation_error", err.Error())
		return
	}
	item, err := h.svc.AddMeal(middleware.CurrentUser(c).ID, req)
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal menyimpan log makanan")
		return
	}
	response.OK(c, 201, item)
}

// POST /logs/water  {"amount_ml":250}  atau  {"reset":true}
func (h *LogHandler) AddWater(c *gin.Context) {
	var req dto.WaterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "validation_error", err.Error())
		return
	}
	uid := middleware.CurrentUser(c).ID
	var (
		res *dto.WaterProgress
		err error
	)
	switch {
	case req.Reset:
		res, err = h.svc.ResetWater(uid)
	case req.AmountML > 0:
		res, err = h.svc.AddWater(uid, req.AmountML)
	default:
		response.Fail(c, 400, "validation_error", "isi amount_ml (>0) atau reset=true")
		return
	}
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal menyimpan log air")
		return
	}
	response.OK(c, 200, res)
}

// GET /logs/history?days=7
func (h *LogHandler) History(c *gin.Context) {
	days := 7
	if v := c.Query("days"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 31 {
			response.Fail(c, 400, "validation_error", "days harus 1-31")
			return
		}
		days = n
	}
	res, err := h.svc.History(middleware.CurrentUser(c).ID, days)
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal mengambil riwayat")
		return
	}
	response.OK(c, 200, res)
}

// GET/POST /favorites
func (h *LogHandler) ListFavorites(c *gin.Context) {
	items, err := h.svc.ListFavorites(middleware.CurrentUser(c).ID)
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal mengambil favorit")
		return
	}
	response.OK(c, 200, items)
}

func (h *LogHandler) AddFavorite(c *gin.Context) {
	var req dto.FavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "validation_error", err.Error())
		return
	}
	item, err := h.svc.AddFavorite(middleware.CurrentUser(c).ID, req)
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal menyimpan favorit")
		return
	}
	response.OK(c, 201, item)
}
