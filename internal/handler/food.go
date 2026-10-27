package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"nutrai/internal/middleware"
	"nutrai/internal/response"
	"nutrai/internal/service"
)

type FoodHandler struct{ svc *service.FoodService }

func NewFoodHandler(s *service.FoodService) *FoodHandler { return &FoodHandler{svc: s} }

var validCategories = map[string]bool{"Breakfast": true, "Lunch": true, "Dinner": true, "Snack": true}

func intQuery(c *gin.Context, key string, def, min, max int) (int, bool) {
	v := c.Query(key)
	if v == "" {
		return def, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min || n > max {
		return 0, false
	}
	return n, true
}

// GET /foods?q=ayam&category=Dinner&tag=high%20protein&safe=true&limit=20&offset=0
// safe (default true) = terapkan filter alergi/halal/diet/budget dari profil user.
func (h *FoodHandler) Search(c *gin.Context) {
	category := c.Query("category")
	if category != "" && !validCategories[category] {
		response.Fail(c, 400, "validation_error", "category harus Breakfast, Lunch, Dinner, atau Snack")
		return
	}
	limit, ok1 := intQuery(c, "limit", 20, 1, 100)
	offset, ok2 := intQuery(c, "offset", 0, 0, 100000)
	if !ok1 || !ok2 {
		response.Fail(c, 400, "validation_error", "limit (1-100) / offset tidak valid")
		return
	}
	safe := strings.ToLower(c.DefaultQuery("safe", "true")) != "false"
	res, err := h.svc.Search(middleware.CurrentUser(c), c.Query("q"), category, c.Query("tag"), safe, limit, offset)
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal mencari makanan")
		return
	}
	response.OK(c, 200, res)
}

// GET /suggestions?meal_type=Dinner&limit=6   (meal_type default: tebakan dari jam lokal)
func (h *FoodHandler) Suggestions(c *gin.Context) {
	mt := c.Query("meal_type")
	if mt != "" && !validCategories[mt] {
		response.Fail(c, 400, "validation_error", "meal_type harus Breakfast, Lunch, Dinner, atau Snack")
		return
	}
	limit, ok := intQuery(c, "limit", 6, 1, 20)
	if !ok {
		response.Fail(c, 400, "validation_error", "limit harus 1-20")
		return
	}
	res, err := h.svc.Suggest(middleware.CurrentUser(c), mt, limit)
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal membuat rekomendasi")
		return
	}
	response.OK(c, 200, res)
}

// GET /ingredients?q=nasi&limit=20  -> data bahan per 100 g (TKPI/USDA) hasil impor.
func (h *FoodHandler) Ingredients(c *gin.Context) {
	limit, ok := intQuery(c, "limit", 20, 1, 50)
	if !ok {
		response.Fail(c, 400, "validation_error", "limit harus 1-50")
		return
	}
	items, err := h.svc.SearchIngredients(c.Query("q"), limit)
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal mencari bahan")
		return
	}
	response.OK(c, 200, items)
}

// GET /foods/:id -> menu + rincian bahan
func (h *FoodHandler) Detail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, 400, "validation_error", "id tidak valid")
		return
	}
	res, err := h.svc.Detail(uint(id))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, 404, "not_found", "menu tidak ditemukan")
		return
	}
	if err != nil {
		response.Fail(c, 500, "internal_error", "gagal mengambil detail menu")
		return
	}
	response.OK(c, 200, res)
}
