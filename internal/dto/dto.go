package dto

import "time"

// ---------- Auth ----------
type RegisterRequest struct {
	Name  string `json:"name" binding:"required,min=2,max=100"`
	Email string `json:"email" binding:"required,email,max=191"`
}

type AuthResponse struct {
	UserID uint   `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	APIKey string `json:"api_key"`
	Note   string `json:"note"`
}

// ---------- Profile ----------
type ProfileResponse struct {
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	CalorieTarget    int      `json:"calorie_target"`
	ProteinTarget    int      `json:"protein_target"`
	CarbsTarget      int      `json:"carbs_target"`
	FatTarget        int      `json:"fat_target"`
	WaterTargetL     float64  `json:"water_target_l"`
	DietPreference   string   `json:"diet_preference"`
	IsHalal          bool     `json:"is_halal"`
	Allergies        []string `json:"allergies"`
	BudgetPreference string   `json:"budget_preference"`
}

// Semua field pointer => partial update (PATCH-like) lewat PUT.
type UpdateProfileRequest struct {
	CalorieTarget    *int      `json:"calorie_target" binding:"omitempty,min=800,max=10000"`
	ProteinTarget    *int      `json:"protein_target" binding:"omitempty,min=0,max=500"`
	CarbsTarget      *int      `json:"carbs_target" binding:"omitempty,min=0,max=1500"`
	FatTarget        *int      `json:"fat_target" binding:"omitempty,min=0,max=500"`
	WaterTargetL     *float64  `json:"water_target_l" binding:"omitempty,min=0.5,max=10"`
	DietPreference   *string   `json:"diet_preference" binding:"omitempty,oneof=none high_protein vegetarian vegan low_lactose"`
	IsHalal          *bool     `json:"is_halal"`
	Allergies        *[]string `json:"allergies" binding:"omitempty,max=20"`
	BudgetPreference *string   `json:"budget_preference" binding:"omitempty,oneof=budget_friendly moderate flexible"`
}

// ---------- Meals / Water ----------
type AddMealRequest struct {
	ItemName       string     `json:"item_name" binding:"required,max=150"`
	MealCategory   string     `json:"meal_category" binding:"required,oneof=Breakfast Lunch Dinner Snack"`
	Calories       int        `json:"calories" binding:"min=0,max=10000"`
	ProteinG       float64    `json:"protein_g" binding:"min=0,max=1000"`
	CarbsG         float64    `json:"carbs_g" binding:"min=0,max=2000"`
	FatG           float64    `json:"fat_g" binding:"min=0,max=1000"`
	EatenAt        *time.Time `json:"eaten_at"` // default: sekarang
	Tags           []string   `json:"tags"`
	SaveAsFavorite bool       `json:"save_as_favorite"`
}

type MealItem struct {
	ID           uint      `json:"id"`
	ItemName     string    `json:"item_name"`
	MealCategory string    `json:"meal_category"`
	Calories     int       `json:"calories"`
	ProteinG     float64   `json:"protein_g"`
	CarbsG       float64   `json:"carbs_g"`
	FatG         float64   `json:"fat_g"`
	EatenAt      time.Time `json:"eaten_at"`
}

type WaterRequest struct {
	AmountML int  `json:"amount_ml" binding:"omitempty,min=1,max=5000"`
	Reset    bool `json:"reset"` // true => hapus semua log air hari ini
}

type FavoriteRequest struct {
	ItemName string   `json:"item_name" binding:"required,max=150"`
	Calories int      `json:"calories" binding:"min=0,max=10000"`
	ProteinG float64  `json:"protein_g" binding:"min=0,max=1000"`
	CarbsG   float64  `json:"carbs_g" binding:"min=0,max=2000"`
	FatG     float64  `json:"fat_g" binding:"min=0,max=1000"`
	Tags     []string `json:"tags"`
}

type FavoriteItem struct {
	ID       uint     `json:"id"`
	ItemName string   `json:"item_name"`
	Calories int      `json:"calories"`
	ProteinG float64  `json:"protein_g"`
	CarbsG   float64  `json:"carbs_g"`
	FatG     float64  `json:"fat_g"`
	Tags     []string `json:"tags"`
}

// ---------- Today / History ----------
type Macros struct {
	Calories int     `json:"calories"`
	ProteinG float64 `json:"protein_g"`
	CarbsG   float64 `json:"carbs_g"`
	FatG     float64 `json:"fat_g"`
}

type Targets struct {
	Calories int     `json:"calories"`
	ProteinG int     `json:"protein_g"`
	CarbsG   int     `json:"carbs_g"`
	FatG     int     `json:"fat_g"`
	WaterL   float64 `json:"water_l"`
}

// CaloriesLeft boleh negatif (artinya melebihi target). Gap lain di-clamp ke 0.
type Remaining struct {
	CaloriesLeft int     `json:"calories_left"`
	ProteinGapG  float64 `json:"protein_gap_g"`
	CarbsGapG    float64 `json:"carbs_gap_g"`
	FatGapG      float64 `json:"fat_gap_g"`
	WaterGapL    float64 `json:"water_gap_l"`
}

type Progress struct {
	Calories int `json:"calories"`
	Protein  int `json:"protein"`
	Carbs    int `json:"carbs"`
	Fat      int `json:"fat"`
	Water    int `json:"water"`
}

type WaterProgress struct {
	TotalML int     `json:"total_ml"`
	TargetL float64 `json:"target_l"`
	LeftML  int     `json:"left_ml"`
}

type TodayResponse struct {
	Date            string                `json:"date"`
	LogCount        int                   `json:"log_count"`
	Meals           []MealItem            `json:"meals"`
	MealsByCategory map[string][]MealItem `json:"meals_by_category"`
	Targets         Targets               `json:"targets"`
	Consumed        Macros                `json:"consumed"`
	Remaining       Remaining             `json:"remaining"`
	ProgressPercent Progress              `json:"progress_percent"`
	Water           WaterProgress         `json:"water"`
}

type HistoryDay struct {
	Date     string  `json:"date"`
	Calories int     `json:"calories"`
	ProteinG float64 `json:"protein_g"`
	WaterML  int     `json:"water_ml"`
	Meals    int     `json:"meals"`
}

type HistoryAverages struct {
	Calories int     `json:"calories"`
	ProteinG float64 `json:"protein_g"`
	WaterL   float64 `json:"water_l"`
}

type HistoryResponse struct {
	Days                int             `json:"days"`
	From                string          `json:"from"`
	To                  string          `json:"to"`
	Daily               []HistoryDay    `json:"daily"`
	Averages            HistoryAverages `json:"averages"`
	TotalMeals          int             `json:"total_meals"`
	DaysNearCalorieGoal int             `json:"days_near_calorie_goal"`
	DaysNearProteinGoal int             `json:"days_near_protein_goal"`
}

// ---------- AI ----------
type ScanResult struct {
	ItemName          string   `json:"item_name"`
	EstimatedCalories float64  `json:"estimated_calories"`
	ProteinG          float64  `json:"protein_g"`
	CarbsG            float64  `json:"carbs_g"`
	FatG              float64  `json:"fat_g"`
	Tags              []string `json:"tags"`
	Confidence        *float64 `json:"confidence,omitempty"`
}

type ChatTurn struct {
	Role    string `json:"role" binding:"required,oneof=user assistant"`
	Content string `json:"content" binding:"required,max=2000"`
}

type ChatRequest struct {
	Message   string     `json:"message" binding:"required,max=1000"`
	Intent    string     `json:"intent" binding:"omitempty,max=50"` // mis. suggest_dinner, explain_day, what_am_i_low_on
	SessionID string     `json:"session_id" binding:"omitempty,max=64"`
	History   []ChatTurn `json:"history" binding:"omitempty,max=10,dive"`
}

type Suggestion struct {
	ItemName string   `json:"item_name"`
	Category string   `json:"category,omitempty"`
	Calories float64  `json:"calories"`
	ProteinG float64  `json:"protein_g"`
	CarbsG   float64  `json:"carbs_g"`
	FatG     float64  `json:"fat_g"`
	Tags     []string `json:"tags,omitempty"`
	Reason   string   `json:"reason,omitempty"`
}

type ChatReply struct {
	Reply       string       `json:"reply"`
	Suggestions []Suggestion `json:"suggestions"`
}

// ----- payload Go -> n8n (AI Chat) -----
type N8NChatPayload struct {
	RequestID string         `json:"request_id"`
	Timestamp time.Time      `json:"timestamp"`
	Message   string         `json:"message"`
	Intent    string         `json:"intent,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	History   []ChatTurn     `json:"history,omitempty"`
	User      N8NUser        `json:"user"`
	Context   N8NChatContext `json:"context"`
}

type N8NUser struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type N8NMealBrief struct {
	ItemName string  `json:"item_name"`
	Category string  `json:"meal_category"`
	Calories int     `json:"calories"`
	ProteinG float64 `json:"protein_g"`
	EatenAt  string  `json:"eaten_at"` // HH:MM lokal
}

type N8NChatContext struct {
	Date         string          `json:"date"`
	LocalTime    string          `json:"local_time"`
	NextMealType string          `json:"next_meal_type"`
	Targets      Targets         `json:"targets"`
	Consumed     Macros          `json:"consumed"`
	Remaining    Remaining       `json:"remaining"`
	Water        WaterProgress   `json:"water"`
	Profile      N8NProfile      `json:"profile"`
	TodayLogs    []N8NMealBrief  `json:"today_logs"`
	// Menu kandidat dari dataset (sudah difilter alergi/halal/diet/budget & diberi peringkat).
	CandidateFoods []SuggestionItem `json:"candidate_foods"`
}

type N8NProfile struct {
	DietPreference   string   `json:"diet_preference"`
	IsHalal          bool     `json:"is_halal"`
	Allergies        []string `json:"allergies"`
	BudgetPreference string   `json:"budget_preference"`
}

// ---------- Foods dataset & suggestions ----------
type FoodItem struct {
	ID       uint     `json:"id"`
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Serving  string   `json:"serving"`
	ServingG int      `json:"serving_g"`
	Calories int      `json:"calories"`
	ProteinG float64  `json:"protein_g"`
	CarbsG   float64  `json:"carbs_g"`
	FatG     float64  `json:"fat_g"`
	Tags     []string `json:"tags"`
	Budget   string   `json:"budget"`
	IsHalal  bool     `json:"is_halal"`
	Source   string   `json:"source"`
	Slug     string   `json:"slug"`
	ImageURL string   `json:"image_url"` // "" bila belum ada gambar -> frontend tampilkan placeholder
}

type FoodListResponse struct {
	Items  []FoodItem `json:"items"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

type SuggestionItem struct {
	FoodItem
	Labels     []string `json:"labels"`
	Reason     string   `json:"reason"`
	Score      float64  `json:"score"`
	IsFavorite bool     `json:"is_favorite"`
}

type SuggestionsResponse struct {
	MealType  string           `json:"meal_type"`
	Remaining Remaining        `json:"remaining"`
	Items     []SuggestionItem `json:"items"`
}

type IngredientItem struct {
	ID              uint    `json:"id"`
	Name            string  `json:"name"`
	CaloriesPer100g float64 `json:"calories_per_100g"`
	ProteinGPer100g float64 `json:"protein_g_per_100g"`
	CarbsGPer100g   float64 `json:"carbs_g_per_100g"`
	FatGPer100g     float64 `json:"fat_g_per_100g"`
	Source          string  `json:"source"`
}

type FoodComponentItem struct {
	Name     string  `json:"name"`
	Grams    float64 `json:"grams"`
	Calories float64 `json:"calories"`
	ProteinG float64 `json:"protein_g"`
	CarbsG   float64 `json:"carbs_g"`
	FatG     float64 `json:"fat_g"`
	Source   string  `json:"source"`
	Verified bool    `json:"verified"`
}

// FoodDetail = menu + rincian per bahan, agar angka bisa ditelusuri.
type FoodDetail struct {
	FoodItem
	Ingredients []FoodComponentItem `json:"ingredients"`
}
