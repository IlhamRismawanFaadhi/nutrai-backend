package models

import "time"

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:100;not null"`
	Email     string `gorm:"size:191;uniqueIndex;not null"`
	APIKey    string `gorm:"column:api_key;size:64;uniqueIndex;not null"` // SHA-256 hash
	CreatedAt time.Time
}

type UserProfile struct {
	UserID           uint    `gorm:"primaryKey"`
	CalorieTarget    int     `gorm:"not null;default:2000"`
	ProteinTarget    int     `gorm:"not null;default:100"`
	CarbsTarget      int     `gorm:"not null;default:250"`
	FatTarget        int     `gorm:"not null;default:65"`
	WaterTarget      float64 `gorm:"type:decimal(4,2);not null;default:2.5"` // liter
	DietPreference   string  `gorm:"size:30;not null;default:none"`
	IsHalal          bool    `gorm:"not null;default:false"`
	AllergiesJSON    string  `gorm:"column:allergies_json;type:json"`
	BudgetPreference string  `gorm:"size:20;not null;default:moderate"`
}

type MealLog struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"not null;index:idx_meal_user_eaten,priority:1"`
	ItemName     string    `gorm:"size:150;not null"`
	MealCategory string    `gorm:"size:20;not null"`
	Calories     int       `gorm:"not null"`
	Protein      float64   `gorm:"type:decimal(6,1);not null"`
	Carbs        float64   `gorm:"type:decimal(6,1);not null"`
	Fat          float64   `gorm:"type:decimal(6,1);not null"`
	EatenAt      time.Time `gorm:"not null;index:idx_meal_user_eaten,priority:2"`
	CreatedAt    time.Time
}

type WaterLog struct {
	ID       uint      `gorm:"primaryKey"`
	UserID   uint      `gorm:"not null;index:idx_water_user_logged,priority:1"`
	AmountML int       `gorm:"column:amount_ml;not null"`
	LoggedAt time.Time `gorm:"not null;index:idx_water_user_logged,priority:2"`
}

type FavoriteMeal struct {
	ID       uint    `gorm:"primaryKey"`
	UserID   uint    `gorm:"not null;index"`
	ItemName string  `gorm:"size:150;not null"`
	Calories int     `gorm:"not null"`
	Protein  float64 `gorm:"type:decimal(6,1);not null"`
	Carbs    float64 `gorm:"type:decimal(6,1);not null"`
	Fat      float64 `gorm:"type:decimal(6,1);not null"`
	TagsJSON string  `gorm:"column:tags_json;type:json"`
}

// FoodItem = dataset makanan (meal library) yang dipakai untuk pencarian & rekomendasi.
type FoodItem struct {
	ID       uint    `gorm:"primaryKey"`
	Name     string  `gorm:"size:150;not null;uniqueIndex"`
	Category string  `gorm:"size:20;not null;index"` // Breakfast/Lunch/Dinner/Snack
	Serving  string  `gorm:"size:80"`
	ServingG int     `gorm:"column:serving_g"` // total gram per porsi (dari komposisi bahan)
	Calories int     `gorm:"not null"`
	Protein  float64 `gorm:"type:decimal(6,1);not null"`
	Carbs    float64 `gorm:"type:decimal(6,1);not null"`
	Fat      float64 `gorm:"type:decimal(6,1);not null"`
	TagsJSON string  `gorm:"column:tags_json;type:json"`
	Budget   string  `gorm:"size:20;not null;default:moderate"` // budget_friendly/moderate/premium
	IsHalal  bool    `gorm:"not null;default:true"`
	Source   string  `gorm:"size:30;not null;default:recipe"` // recipe (dihitung dari bahan) | estimate | tkpi | custom
	Slug        string `gorm:"size:160;index"`  // dasar nama file gambar: <slug>.jpg
	ImageURL    string `gorm:"size:500"`        // override URL gambar (opsional)
	ImageCredit string `gorm:"size:200"`        // atribusi/lisensi gambar (opsional)
}

func (FoodItem) TableName() string { return "foods" }

// Ingredient = data bahan pangan per 100 g dari sumber rujukan (TKPI / USDA / custom).
type Ingredient struct {
	ID          uint    `gorm:"primaryKey"`
	Name        string  `gorm:"size:200;not null;index"`
	Calories100 float64 `gorm:"column:calories_100g;type:decimal(7,2);not null"`
	Protein100  float64 `gorm:"column:protein_100g;type:decimal(7,2);not null"`
	Carbs100    float64 `gorm:"column:carbs_100g;type:decimal(7,2);not null"`
	Fat100      float64 `gorm:"column:fat_100g;type:decimal(7,2);not null"`
	Source      string  `gorm:"size:20;not null;index"` // tkpi | usda | custom
	SourceRef   string  `gorm:"size:100"`               // id baris di dataset asal
	Code        *string `gorm:"column:code;size:50;uniqueIndex"` // kunci seed (NULL untuk hasil impor CSV)
	State       string  `gorm:"size:20"`                         // raw | cooked | dry | as_eaten
	Verified    bool    `gorm:"not null;default:false"`          // true = sudah direview ahli gizi; seed tidak menimpa
	Note        string  `gorm:"size:300"`
}

func (Ingredient) TableName() string { return "ingredients" }

// FoodComponent = resep: menu (foods) terdiri dari bahan (ingredients) sekian gram.
// Gizi menu dihitung dari komponen ini agar angkanya bisa ditelusuri ke sumber.
type FoodComponent struct {
	ID           uint    `gorm:"primaryKey"`
	FoodID       uint    `gorm:"not null;uniqueIndex:uq_food_ingredient,priority:1"`
	IngredientID uint    `gorm:"not null;uniqueIndex:uq_food_ingredient,priority:2"`
	Grams        float64 `gorm:"type:decimal(7,1);not null"`
}

func (FoodComponent) TableName() string { return "food_components" }
