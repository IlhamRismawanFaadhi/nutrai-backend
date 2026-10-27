package database

import (
	"math"

	"gorm.io/gorm"

	"nutrai/internal/models"
)

type compRow struct {
	Grams    float64 `gorm:"column:grams"`
	Calories float64 `gorm:"column:calories"`
	Protein  float64 `gorm:"column:protein"`
	Carbs    float64 `gorm:"column:carbs"`
	Fat      float64 `gorm:"column:fat"`
}

// RecomputeFood menghitung ulang kalori/makro/serving_g satu menu dari komposisi bahan di DB
// (jumlah gram x nilai per 100 g / 100). Menu tanpa komponen dibiarkan.
// Dipakai oleh seed dan importer agar angka menu selalu = tabel bahan saat ini.
func RecomputeFood(db *gorm.DB, foodID uint) error {
	var rows []compRow
	err := db.Table("food_components AS fc").
		Select("fc.grams AS grams, i.calories_100g AS calories, i.protein_100g AS protein, i.carbs_100g AS carbs, i.fat_100g AS fat").
		Joins("JOIN ingredients AS i ON i.id = fc.ingredient_id").
		Where("fc.food_id = ?", foodID).Scan(&rows).Error
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	var cal, p, c, f, g float64
	for _, r := range rows {
		k := r.Grams / 100
		cal += r.Calories * k
		p += r.Protein * k
		c += r.Carbs * k
		f += r.Fat * k
		g += r.Grams
	}
	r1 := func(v float64) float64 { return math.Round(v*10) / 10 }
	return db.Model(&models.FoodItem{}).Where("id = ?", foodID).Updates(map[string]any{
		"calories": int(math.Round(cal)), "protein": r1(p), "carbs": r1(c), "fat": r1(f),
		"serving_g": int(math.Round(g)),
	}).Error
}
