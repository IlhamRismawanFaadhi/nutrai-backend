package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"nutrai/internal/dto"
	"nutrai/internal/models"
	"nutrai/internal/repository"
)

type FoodService struct {
	repo     *repository.Repository
	profiles *ProfileService
	logs     *LogService
	loc      *time.Location
	imgBase  string
}

func NewFoodService(repo *repository.Repository, p *ProfileService, l *LogService, loc *time.Location, imageBaseURL string) *FoodService {
	return &FoodService{repo: repo, profiles: p, logs: l, loc: loc, imgBase: imageBaseURL}
}

// ---------- filter profil user (hard filter) ----------
type userFilter struct {
	allergies map[string]bool
	halal     bool
	diet      string
	budget    string
}

func newFilter(p *dto.ProfileResponse) userFilter {
	f := userFilter{allergies: map[string]bool{}, halal: p.IsHalal, diet: p.DietPreference, budget: p.BudgetPreference}
	for _, a := range p.Allergies {
		f.allergies[canonicalAllergy(a)] = true
	}
	return f
}

func canonicalAllergy(a string) string {
	a = strings.ToLower(strings.TrimSpace(a))
	switch a {
	case "peanuts":
		return "peanut"
	case "eggs":
		return "egg"
	case "nuts", "tree nut":
		return "tree nuts"
	}
	return a
}

func hasTag(tags []string, t string) bool {
	for _, x := range tags {
		if x == t {
			return true
		}
	}
	return false
}

func (f userFilter) allows(it models.FoodItem, tags []string) bool {
	for _, t := range tags {
		if f.allergies[t] {
			return false
		}
	}
	if f.halal && !it.IsHalal {
		return false
	}
	switch f.diet {
	case "vegetarian":
		if !hasTag(tags, "vegetarian") {
			return false
		}
	case "vegan":
		if !hasTag(tags, "vegan") {
			return false
		}
	case "low_lactose":
		if hasTag(tags, "dairy") {
			return false
		}
	}
	switch f.budget {
	case "budget_friendly":
		return it.Budget == "budget_friendly"
	case "moderate":
		return it.Budget != "premium"
	}
	return true
}

// imageURL: kolom image_url (override) > FOOD_IMAGE_BASE_URL/<slug>.jpg > "".
func (s *FoodService) imageURL(f models.FoodItem) string {
	if f.ImageURL != "" {
		return f.ImageURL
	}
	if s.imgBase != "" && f.Slug != "" {
		return s.imgBase + "/" + f.Slug + ".jpg"
	}
	return ""
}

func (s *FoodService) toFoodDTO(f models.FoodItem) dto.FoodItem {
	return dto.FoodItem{Slug: f.Slug, ImageURL: s.imageURL(f), ServingG: f.ServingG, ID: f.ID, Name: f.Name, Category: f.Category, Serving: f.Serving, Calories: f.Calories,
		ProteinG: f.Protein, CarbsG: f.Carbs, FatG: f.Fat, Tags: jsonToTags(f.TagsJSON),
		Budget: f.Budget, IsHalal: f.IsHalal, Source: f.Source}
}

// ---------- Search ----------
// safe=true => terapkan filter alergi/halal/diet/budget dari profil user.
func (s *FoodService) Search(u *models.User, q, category, tag string, safe bool, limit, offset int) (*dto.FoodListResponse, error) {
	rows, err := s.repo.SearchFoods(strings.TrimSpace(q), category, strings.ToLower(strings.TrimSpace(tag)), 1000)
	if err != nil {
		return nil, err
	}
	var flt userFilter
	if safe {
		prof, err := s.profiles.Get(u)
		if err != nil {
			return nil, err
		}
		flt = newFilter(prof)
	}
	items := make([]dto.FoodItem, 0, len(rows))
	for _, r := range rows {
		if safe && !flt.allows(r, jsonToTags(r.TagsJSON)) {
			continue
		}
		items = append(items, s.toFoodDTO(r))
	}
	total := len(items)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return &dto.FoodListResponse{Items: items[offset:end], Total: total, Limit: limit, Offset: offset}, nil
}

// ---------- Suggest (ranking berbasis aturan, tanpa LLM) ----------
var mealShare = map[string]float64{"Breakfast": 0.40, "Lunch": 0.50, "Dinner": 0.70, "Snack": 0.20}

func isMain(c string) bool { return c == "Lunch" || c == "Dinner" }

func (s *FoodService) Suggest(u *models.User, mealType string, limit int) (*dto.SuggestionsResponse, error) {
	now := time.Now().In(s.loc)
	if mealType == "" {
		mealType = nextMealType(now.Hour())
	}
	if limit < 1 {
		limit = 6
	}
	prof, err := s.profiles.Get(u)
	if err != nil {
		return nil, err
	}
	today, err := s.logs.Today(u.ID, now)
	if err != nil {
		return nil, err
	}
	favs, err := s.repo.ListFavorites(u.ID)
	if err != nil {
		return nil, err
	}
	favSet := map[string]bool{}
	for _, f := range favs {
		favSet[strings.ToLower(f.ItemName)] = true
	}
	foods, err := s.repo.SearchFoods("", "", "", 1000)
	if err != nil {
		return nil, err
	}

	flt := newFilter(prof)
	left := float64(today.Remaining.CaloriesLeft)
	pGap, cGap, fGap := today.Remaining.ProteinGapG, today.Remaining.CarbsGapG, today.Remaining.FatGapG

	type cand struct {
		item models.FoodItem
		tags []string
	}
	var pool []cand
	for _, f := range foods {
		if f.Category != mealType && !(isMain(mealType) && isMain(f.Category)) {
			continue
		}
		tags := jsonToTags(f.TagsJSON)
		if !flt.allows(f, tags) {
			continue
		}
		pool = append(pool, cand{f, tags})
	}
	// Batasi ke sisa kalori; bila tidak ada yang muat, pakai seluruh pool (skor akan memilih yang teringan).
	var fit []cand
	for _, c := range pool {
		if left > 0 && float64(c.item.Calories) <= left {
			fit = append(fit, c)
		}
	}
	if len(fit) == 0 {
		fit = pool
	}

	target := left * mealShare[mealType]
	if target < 150 {
		target = 150
	}
	focus := "balanced intake"
	if pGap >= 15 {
		focus = "your protein goal"
	}
	meal := strings.ToLower(mealType)

	out := make([]dto.SuggestionItem, 0, len(fit))
	for _, c := range fit {
		it := c.item
		cal := float64(it.Calories)

		calScore := math.Max(0, 1-math.Abs(cal-target)/target) * 40
		protScore := 17.5
		if pGap > 0 {
			protScore = math.Min(it.Protein/pGap, 1) * 35
		}
		penalty := 0.0
		if cGap > 0 && it.Carbs > cGap*1.2 {
			penalty += 7
		}
		if fGap > 0 && it.Fat > fGap*1.2 {
			penalty += 8
		}
		score := calScore + protScore - penalty
		if it.Category == mealType {
			score += 8
		}
		isFav := favSet[strings.ToLower(it.Name)]
		if isFav {
			score += 10
		}
		if prof.DietPreference == "high_protein" && hasTag(c.tags, "high protein") {
			score += 5
		}

		labels := []string{fmt.Sprintf("Best %s fit", meal)}
		if pGap > 0 && (it.Protein >= pGap*0.6 || it.Protein >= 35) {
			labels = append(labels, "Protein fit")
		}
		if cal <= 480 || (left > 0 && cal <= left*0.6) {
			labels = append(labels, "Light option")
		}
		reason := fmt.Sprintf("Ranked for %s toward %s: %d kcal fits the remaining calories; %s matches the next likely meal.",
			focus, meal, it.Calories, meal)
		if left > 0 && cal > left {
			reason = fmt.Sprintf("Ranked for %s toward %s: %d kcal is a bit above the remaining calories.", focus, meal, it.Calories)
		}

		out = append(out, dto.SuggestionItem{FoodItem: s.toFoodDTO(it), Labels: labels, Reason: reason,
			Score: round1(score), IsFavorite: isFav})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Name < out[j].Name
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return &dto.SuggestionsResponse{MealType: mealType, Remaining: today.Remaining, Items: out}, nil
}

func (s *FoodService) SearchIngredients(q string, limit int) ([]dto.IngredientItem, error) {
	rows, err := s.repo.SearchIngredients(strings.TrimSpace(q), limit)
	if err != nil {
		return nil, err
	}
	out := make([]dto.IngredientItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.IngredientItem{ID: r.ID, Name: r.Name, CaloriesPer100g: r.Calories100,
			ProteinGPer100g: r.Protein100, CarbsGPer100g: r.Carbs100, FatGPer100g: r.Fat100, Source: r.Source})
	}
	return out, nil
}

// Detail mengembalikan menu + rincian bahan (gram & kontribusi gizi tiap bahan).
func (s *FoodService) Detail(id uint) (*dto.FoodDetail, error) {
	f, err := s.repo.GetFood(id)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.FoodComponents(id)
	if err != nil {
		return nil, err
	}
	items := make([]dto.FoodComponentItem, 0, len(rows))
	for _, r := range rows {
		k := r.Grams / 100
		items = append(items, dto.FoodComponentItem{Name: r.Name, Grams: r.Grams,
			Calories: round1(r.Calories * k), ProteinG: round1(r.Protein * k),
			CarbsG: round1(r.Carbs * k), FatG: round1(r.Fat * k), Source: r.Source, Verified: r.Verified})
	}
	return &dto.FoodDetail{FoodItem: s.toFoodDTO(*f), Ingredients: items}, nil
}
