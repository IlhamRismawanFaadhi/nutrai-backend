package repository

import (
	"time"

	"gorm.io/gorm"

	"nutrai/internal/models"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Ping() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// ---------- users ----------
func (r *Repository) EmailExists(email string) (bool, error) {
	var n int64
	err := r.db.Model(&models.User{}).Where("email = ?", email).Count(&n).Error
	return n > 0, err
}

func (r *Repository) CreateUserWithProfile(u *models.User, p *models.UserProfile) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		p.UserID = u.ID
		return tx.Create(p).Error
	})
}

func (r *Repository) FindUserByKeyHash(hash string) (*models.User, error) {
	var u models.User
	if err := r.db.Where("api_key = ?", hash).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) UpdateAPIKey(userID uint, hash string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("api_key", hash).Error
}

// ---------- profile ----------
func (r *Repository) GetProfile(userID uint) (*models.UserProfile, error) {
	var p models.UserProfile
	if err := r.db.Where("user_id = ?", userID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) SaveProfile(p *models.UserProfile) error { return r.db.Save(p).Error }

// ---------- meals ----------
func (r *Repository) CreateMeal(m *models.MealLog) error { return r.db.Create(m).Error }

func (r *Repository) MealsBetween(userID uint, from, to time.Time) ([]models.MealLog, error) {
	var out []models.MealLog
	err := r.db.Where("user_id = ? AND eaten_at >= ? AND eaten_at < ?", userID, from, to).
		Order("eaten_at ASC").Find(&out).Error
	return out, err
}

// ---------- water ----------
func (r *Repository) CreateWater(w *models.WaterLog) error { return r.db.Create(w).Error }

func (r *Repository) WaterBetween(userID uint, from, to time.Time) ([]models.WaterLog, error) {
	var out []models.WaterLog
	err := r.db.Where("user_id = ? AND logged_at >= ? AND logged_at < ?", userID, from, to).Find(&out).Error
	return out, err
}

func (r *Repository) DeleteWaterBetween(userID uint, from, to time.Time) error {
	return r.db.Where("user_id = ? AND logged_at >= ? AND logged_at < ?", userID, from, to).
		Delete(&models.WaterLog{}).Error
}

// ---------- favorites ----------
func (r *Repository) CreateFavorite(f *models.FavoriteMeal) error { return r.db.Create(f).Error }

func (r *Repository) ListFavorites(userID uint) ([]models.FavoriteMeal, error) {
	var out []models.FavoriteMeal
	err := r.db.Where("user_id = ?", userID).Order("id DESC").Find(&out).Error
	return out, err
}

// ---------- foods (dataset) ----------
func (r *Repository) SearchFoods(q, category, tag string, limit int) ([]models.FoodItem, error) {
	tx := r.db.Model(&models.FoodItem{})
	if q != "" {
		like := "%" + q + "%"
		tx = tx.Where("name LIKE ? OR CAST(tags_json AS CHAR) LIKE ?", like, like)
	}
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	if tag != "" {
		tx = tx.Where("JSON_CONTAINS(tags_json, JSON_QUOTE(?))", tag)
	}
	var out []models.FoodItem
	err := tx.Order("name ASC").Limit(limit).Find(&out).Error
	return out, err
}

// ---------- ingredients ----------
func (r *Repository) SearchIngredients(q string, limit int) ([]models.Ingredient, error) {
	tx := r.db.Model(&models.Ingredient{})
	if q != "" {
		tx = tx.Where("name LIKE ?", "%"+q+"%")
	}
	var out []models.Ingredient
	err := tx.Order("CHAR_LENGTH(name) ASC, name ASC").Limit(limit).Find(&out).Error // nama pendek = bahan dasar dulu
	return out, err
}

func (r *Repository) GetFood(id uint) (*models.FoodItem, error) {
	var f models.FoodItem
	if err := r.db.First(&f, id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

// FoodComponentRow = bahan + gram + nilai per 100 g untuk satu menu.
type FoodComponentRow struct {
	Name       string  `gorm:"column:name"`
	Grams      float64 `gorm:"column:grams"`
	Calories   float64 `gorm:"column:calories_100g"`
	Protein    float64 `gorm:"column:protein_100g"`
	Carbs      float64 `gorm:"column:carbs_100g"`
	Fat        float64 `gorm:"column:fat_100g"`
	Source     string  `gorm:"column:source"`
	Verified   bool    `gorm:"column:verified"`
}

func (r *Repository) FoodComponents(foodID uint) ([]FoodComponentRow, error) {
	var out []FoodComponentRow
	err := r.db.Table("food_components AS fc").
		Select("i.name, fc.grams, i.calories_100g, i.protein_100g, i.carbs_100g, i.fat_100g, i.source, i.verified").
		Joins("JOIN ingredients AS i ON i.id = fc.ingredient_id").
		Where("fc.food_id = ?", foodID).Order("fc.grams DESC").Scan(&out).Error
	return out, err
}
