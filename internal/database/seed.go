package database

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"

	"nutrai/internal/models"
	"nutrai/internal/seed"
)

// SeedFoods idempotent.
//   - Bahan: baris dengan verified=true TIDAK ditimpa (nilai hasil review ahli gizi aman).
//   - Menu: hanya baris ber-source "estimate"/"recipe" yang diperbarui; "tkpi"/"custom" tidak disentuh.
func SeedFoods(db *gorm.DB) error {
	var ings []seed.Ingredient
	if err := json.Unmarshal(seed.IngredientsJSON, &ings); err != nil {
		return fmt.Errorf("ingredients.json: %w", err)
	}
	var foods []seed.Food
	if err := json.Unmarshal(seed.FoodsJSON, &foods); err != nil {
		return fmt.Errorf("foods.json: %w", err)
	}

	idByKey := map[string]uint{}
	for _, in := range ings {
		var row models.Ingredient
		err := db.Where("code = ?", in.Key).First(&row).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			code := in.Key
			row = models.Ingredient{Code: &code, Name: in.Name, State: in.State, Calories100: in.Kcal100g,
				Protein100: in.Protein100g, Carbs100: in.Carbs100g, Fat100: in.Fat100g, Source: in.Source, Note: in.Note}
			if err := db.Create(&row).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		case !row.Verified:
			if err := db.Model(&row).Updates(map[string]any{"name": in.Name, "state": in.State,
				"calories_100g": in.Kcal100g, "protein_100g": in.Protein100g, "carbs_100g": in.Carbs100g,
				"fat_100g": in.Fat100g, "source": in.Source, "note": in.Note}).Error; err != nil {
				return err
			}
		}
		idByKey[in.Key] = row.ID
	}

	created, updated := 0, 0
	for _, f := range foods {
		tags, _ := json.Marshal(f.Tags)
		var row models.FoodItem
		err := db.Where("name = ?", f.Name).First(&row).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			row = models.FoodItem{Name: f.Name, Category: f.Category, Serving: f.Serving, ServingG: f.ServingG,
				Calories: f.Calories, Protein: f.ProteinG, Carbs: f.CarbsG, Fat: f.FatG, TagsJSON: string(tags),
				Budget: f.Budget, IsHalal: f.IsHalal, Source: "recipe", Slug: slugify(f.Name)}
			if err := db.Create(&row).Error; err != nil {
				return err
			}
			created++
		case err != nil:
			return err
		case row.Source == "estimate" || row.Source == "recipe":
			updates := map[string]any{"category": f.Category, "serving": f.Serving, "serving_g": f.ServingG,
				"calories": f.Calories, "protein": f.ProteinG, "carbs": f.CarbsG, "fat": f.FatG,
				"tags_json": string(tags), "budget": f.Budget, "is_halal": f.IsHalal, "source": "recipe"}
			if row.Slug == "" {
				updates["slug"] = slugify(f.Name)
			}
			if err := db.Model(&row).Updates(updates).Error; err != nil {
				return err
			}
			updated++
		default:
			continue // tkpi / custom: jangan disentuh
		}

		// ganti komposisi bahan
		if err := db.Where("food_id = ?", row.ID).Delete(&models.FoodComponent{}).Error; err != nil {
			return err
		}
		for _, fi := range f.Ingredients {
			id, ok := idByKey[fi.Key]
			if !ok {
				return fmt.Errorf("menu %q memakai bahan tak dikenal %q", f.Name, fi.Key)
			}
			if err := db.Create(&models.FoodComponent{FoodID: row.ID, IngredientID: id, Grams: fi.Grams}).Error; err != nil {
				return err
			}
		}
		// angka menu selalu mengikuti tabel bahan di DB (termasuk bahan yang sudah verified/diedit)
		if err := RecomputeFood(db, row.ID); err != nil {
			return err
		}
	}
	log.Printf("seed: %d bahan, %d menu baru, %d menu diperbarui", len(ings), created, updated)
	return nil
}

func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
