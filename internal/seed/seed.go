// Package seed menyimpan dataset awal (di-embed ke binary). Dibuat oleh tools/gen_foods.py.
package seed

import _ "embed"

//go:embed foods.json
var FoodsJSON []byte

//go:embed ingredients.json
var IngredientsJSON []byte

type Ingredient struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	State       string  `json:"state"`
	Kcal100g    float64 `json:"kcal_100g"`
	Protein100g float64 `json:"protein_100g"`
	Carbs100g   float64 `json:"carbs_100g"`
	Fat100g     float64 `json:"fat_100g"`
	Source      string  `json:"source"`
	Note        string  `json:"note"`
}

type FoodIngredient struct {
	Key   string  `json:"key"`
	Grams float64 `json:"grams"`
}

type Food struct {
	Name        string           `json:"name"`
	Category    string           `json:"category"`
	Serving     string           `json:"serving"`
	ServingG    int              `json:"serving_g"`
	Calories    int              `json:"calories"`
	ProteinG    float64          `json:"protein_g"`
	CarbsG      float64          `json:"carbs_g"`
	FatG        float64          `json:"fat_g"`
	Tags        []string         `json:"tags"`
	Budget      string           `json:"budget"`
	IsHalal     bool             `json:"is_halal"`
	Ingredients []FoodIngredient `json:"ingredients"`
}
