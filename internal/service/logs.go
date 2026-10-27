package service

import (
	"math"
	"strings"
	"time"

	"nutrai/internal/dto"
	"nutrai/internal/models"
	"nutrai/internal/repository"
)

type LogService struct {
	repo *repository.Repository
	loc  *time.Location
}

func NewLogService(repo *repository.Repository, loc *time.Location) *LogService {
	return &LogService{repo: repo, loc: loc}
}

var categoryKeys = []string{"breakfast", "lunch", "dinner", "snack"}

// ParseDate: "" => hari ini; format YYYY-MM-DD (zona waktu aplikasi).
func (s *LogService) ParseDate(v string) (time.Time, error) {
	if v == "" {
		return time.Now().In(s.loc), nil
	}
	return time.ParseInLocation("2006-01-02", v, s.loc)
}

// dayBounds mengembalikan [00:00, 24:00) hari tsb dalam UTC.
func (s *LogService) dayBounds(day time.Time) (time.Time, time.Time) {
	d := day.In(s.loc)
	start := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, s.loc)
	return start.UTC(), start.AddDate(0, 0, 1).UTC()
}

func (s *LogService) toMealItem(m models.MealLog) dto.MealItem {
	return dto.MealItem{
		ID: m.ID, ItemName: m.ItemName, MealCategory: m.MealCategory, Calories: m.Calories,
		ProteinG: m.Protein, CarbsG: m.Carbs, FatG: m.Fat, EatenAt: m.EatenAt.In(s.loc),
	}
}

// ---------- Today ----------
func (s *LogService) Today(userID uint, day time.Time) (*dto.TodayResponse, error) {
	from, to := s.dayBounds(day)
	prof, err := s.repo.GetProfile(userID)
	if err != nil {
		return nil, err
	}
	meals, err := s.repo.MealsBetween(userID, from, to)
	if err != nil {
		return nil, err
	}
	water, err := s.repo.WaterBetween(userID, from, to)
	if err != nil {
		return nil, err
	}

	byCat := map[string][]dto.MealItem{}
	for _, k := range categoryKeys {
		byCat[k] = []dto.MealItem{}
	}
	items := make([]dto.MealItem, 0, len(meals))
	var cal int
	var p, c, f float64
	for _, m := range meals {
		it := s.toMealItem(m)
		items = append(items, it)
		key := strings.ToLower(m.MealCategory)
		byCat[key] = append(byCat[key], it)
		cal += m.Calories
		p += m.Protein
		c += m.Carbs
		f += m.Fat
	}
	waterML := 0
	for _, w := range water {
		waterML += w.AmountML
	}
	p, c, f = round1(p), round1(c), round1(f)

	targetWaterML := prof.WaterTarget * 1000
	left := int(math.Max(0, targetWaterML-float64(waterML)))

	return &dto.TodayResponse{
		Date:            day.In(s.loc).Format("2006-01-02"),
		LogCount:        len(items),
		Meals:           items,
		MealsByCategory: byCat,
		Targets: dto.Targets{
			Calories: prof.CalorieTarget, ProteinG: prof.ProteinTarget, CarbsG: prof.CarbsTarget,
			FatG: prof.FatTarget, WaterL: prof.WaterTarget,
		},
		Consumed: dto.Macros{Calories: cal, ProteinG: p, CarbsG: c, FatG: f},
		Remaining: dto.Remaining{
			CaloriesLeft: prof.CalorieTarget - cal,
			ProteinGapG:  round1(clamp0(float64(prof.ProteinTarget) - p)),
			CarbsGapG:    round1(clamp0(float64(prof.CarbsTarget) - c)),
			FatGapG:      round1(clamp0(float64(prof.FatTarget) - f)),
			WaterGapL:    round2(float64(left) / 1000),
		},
		ProgressPercent: dto.Progress{
			Calories: pct(float64(cal), float64(prof.CalorieTarget)),
			Protein:  pct(p, float64(prof.ProteinTarget)),
			Carbs:    pct(c, float64(prof.CarbsTarget)),
			Fat:      pct(f, float64(prof.FatTarget)),
			Water:    pct(float64(waterML), targetWaterML),
		},
		Water: dto.WaterProgress{TotalML: waterML, TargetL: prof.WaterTarget, LeftML: left},
	}, nil
}

// ---------- Meals ----------
func (s *LogService) AddMeal(userID uint, req dto.AddMealRequest) (*dto.MealItem, error) {
	eaten := time.Now()
	if req.EatenAt != nil {
		eaten = *req.EatenAt
	}
	m := &models.MealLog{
		UserID: userID, ItemName: strings.TrimSpace(req.ItemName), MealCategory: req.MealCategory,
		Calories: req.Calories, Protein: round1(req.ProteinG), Carbs: round1(req.CarbsG), Fat: round1(req.FatG),
		EatenAt: eaten.UTC(),
	}
	if err := s.repo.CreateMeal(m); err != nil {
		return nil, err
	}
	if req.SaveAsFavorite {
		if _, err := s.AddFavorite(userID, dto.FavoriteRequest{
			ItemName: m.ItemName, Calories: m.Calories, ProteinG: m.Protein, CarbsG: m.Carbs, FatG: m.Fat, Tags: req.Tags,
		}); err != nil {
			return nil, err
		}
	}
	it := s.toMealItem(*m)
	return &it, nil
}

// ---------- Water ----------
func (s *LogService) AddWater(userID uint, ml int) (*dto.WaterProgress, error) {
	if err := s.repo.CreateWater(&models.WaterLog{UserID: userID, AmountML: ml, LoggedAt: time.Now().UTC()}); err != nil {
		return nil, err
	}
	return s.waterToday(userID)
}

func (s *LogService) ResetWater(userID uint) (*dto.WaterProgress, error) {
	from, to := s.dayBounds(time.Now())
	if err := s.repo.DeleteWaterBetween(userID, from, to); err != nil {
		return nil, err
	}
	return s.waterToday(userID)
}

func (s *LogService) waterToday(userID uint) (*dto.WaterProgress, error) {
	t, err := s.Today(userID, time.Now())
	if err != nil {
		return nil, err
	}
	return &t.Water, nil
}

// ---------- Favorites ----------
func (s *LogService) AddFavorite(userID uint, req dto.FavoriteRequest) (*dto.FavoriteItem, error) {
	f := &models.FavoriteMeal{
		UserID: userID, ItemName: strings.TrimSpace(req.ItemName), Calories: req.Calories,
		Protein: round1(req.ProteinG), Carbs: round1(req.CarbsG), Fat: round1(req.FatG),
		TagsJSON: tagsToJSON(req.Tags),
	}
	if err := s.repo.CreateFavorite(f); err != nil {
		return nil, err
	}
	return toFavDTO(*f), nil
}

func (s *LogService) ListFavorites(userID uint) ([]dto.FavoriteItem, error) {
	rows, err := s.repo.ListFavorites(userID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.FavoriteItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, *toFavDTO(r))
	}
	return out, nil
}

func toFavDTO(f models.FavoriteMeal) *dto.FavoriteItem {
	return &dto.FavoriteItem{ID: f.ID, ItemName: f.ItemName, Calories: f.Calories,
		ProteinG: f.Protein, CarbsG: f.Carbs, FatG: f.Fat, Tags: jsonToTags(f.TagsJSON)}
}

// ---------- History ----------
func (s *LogService) History(userID uint, days int) (*dto.HistoryResponse, error) {
	if days < 1 {
		days = 7
	}
	now := time.Now().In(s.loc)
	firstDay := now.AddDate(0, 0, -(days - 1))
	from, _ := s.dayBounds(firstDay)
	_, to := s.dayBounds(now)

	prof, err := s.repo.GetProfile(userID)
	if err != nil {
		return nil, err
	}
	meals, err := s.repo.MealsBetween(userID, from, to)
	if err != nil {
		return nil, err
	}
	water, err := s.repo.WaterBetween(userID, from, to)
	if err != nil {
		return nil, err
	}

	const layout = "2006-01-02"
	buckets := map[string]*dto.HistoryDay{}
	order := make([]string, 0, days)
	for i := 0; i < days; i++ {
		d := firstDay.AddDate(0, 0, i).Format(layout)
		buckets[d] = &dto.HistoryDay{Date: d}
		order = append(order, d)
	}
	for _, m := range meals {
		if b := buckets[m.EatenAt.In(s.loc).Format(layout)]; b != nil {
			b.Calories += m.Calories
			b.ProteinG += m.Protein
			b.Meals++
		}
	}
	for _, w := range water {
		if b := buckets[w.LoggedAt.In(s.loc).Format(layout)]; b != nil {
			b.WaterML += w.AmountML
		}
	}

	res := &dto.HistoryResponse{Days: days, From: order[0], To: order[len(order)-1], Daily: make([]dto.HistoryDay, 0, days)}
	var sumCal, sumWater, totalMeals int
	var sumProt float64
	for _, d := range order {
		b := buckets[d]
		b.ProteinG = round1(b.ProteinG)
		res.Daily = append(res.Daily, *b)
		sumCal += b.Calories
		sumProt += b.ProteinG
		sumWater += b.WaterML
		totalMeals += b.Meals
		if near(float64(b.Calories), float64(prof.CalorieTarget)) {
			res.DaysNearCalorieGoal++
		}
		if near(b.ProteinG, float64(prof.ProteinTarget)) {
			res.DaysNearProteinGoal++
		}
	}
	n := float64(days)
	res.Averages = dto.HistoryAverages{
		Calories: int(math.Round(float64(sumCal) / n)),
		ProteinG: round1(sumProt / n),
		WaterL:   round1(float64(sumWater) / n / 1000),
	}
	res.TotalMeals = totalMeals
	return res, nil
}

// near: dalam rentang ±10% dari target (dan ada data).
func near(v, target float64) bool {
	return v > 0 && target > 0 && math.Abs(v-target) <= target*0.10
}
