package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"nutrai/internal/config"
	"nutrai/internal/dto"
	"nutrai/internal/models"
	"nutrai/internal/n8n"
)

var (
	ErrAINotConfigured = errors.New("n8n webhook not configured")
	ErrNoFoodDetected  = errors.New("no food detected in image")
	ErrAIBadResponse   = errors.New("invalid response from AI workflow")
)

type AIService struct {
	n8n      *n8n.Client
	cfg      *config.Config
	profiles *ProfileService
	logs     *LogService
	foods    *FoodService
}

func NewAIService(c *n8n.Client, cfg *config.Config, p *ProfileService, l *LogService, f *FoodService) *AIService {
	return &AIService{n8n: c, cfg: cfg, profiles: p, logs: l, foods: f}
}

// ---------- AI Scan ----------
func (s *AIService) Scan(ctx context.Context, u *models.User, filename, contentType string, data []byte) (*dto.ScanResult, error) {
	if s.cfg.N8NScanURL == "" {
		return nil, ErrAINotConfigured
	}
	prof, err := s.profiles.Get(u)
	if err != nil {
		return nil, err
	}
	// Konteks profil ikut dikirim agar n8n bisa menandai tag allergen / halal.
	allergies, _ := json.Marshal(prof.Allergies)
	fields := map[string]string{
		"request_id": newRequestID(),
		"user_id":    fmt.Sprint(u.ID),
		"allergies":  string(allergies),
		"is_halal":   fmt.Sprint(prof.IsHalal),
		"diet":       prof.DietPreference,
	}

	var raw json.RawMessage
	if err := s.n8n.PostMultipart(ctx, s.cfg.N8NScanURL, fields, "image", filename, contentType, data, &raw); err != nil {
		return nil, err
	}
	var res dto.ScanResult
	if err := json.Unmarshal(unwrap(raw), &res); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIBadResponse, err)
	}
	res.ItemName = strings.TrimSpace(res.ItemName)
	if res.ItemName == "" {
		return nil, ErrNoFoodDetected
	}
	res.Tags = normalizeTags(res.Tags)
	res.EstimatedCalories = float64(int(res.EstimatedCalories + 0.5))
	res.ProteinG, res.CarbsG, res.FatG = round1(res.ProteinG), round1(res.CarbsG), round1(res.FatG)
	return &res, nil
}

// ---------- AI Chat ----------
func (s *AIService) Chat(ctx context.Context, u *models.User, req dto.ChatRequest) (*dto.ChatReply, error) {
	if s.cfg.N8NChatURL == "" {
		return nil, ErrAINotConfigured
	}
	now := time.Now().In(s.cfg.Location)
	today, err := s.logs.Today(u.ID, now)
	if err != nil {
		return nil, err
	}
	prof, err := s.profiles.Get(u)
	if err != nil {
		return nil, err
	}

	// Kandidat menu dari dataset agar jawaban AI berbasis data nyata (bukan karangan model).
	sug, err := s.foods.Suggest(u, "", 8)
	if err != nil {
		return nil, err
	}

	briefs := make([]dto.N8NMealBrief, 0, len(today.Meals))
	for _, m := range today.Meals {
		briefs = append(briefs, dto.N8NMealBrief{
			ItemName: m.ItemName, Category: m.MealCategory, Calories: m.Calories,
			ProteinG: m.ProteinG, EatenAt: m.EatenAt.Format("15:04"),
		})
	}
	payload := dto.N8NChatPayload{
		RequestID: newRequestID(), Timestamp: time.Now().UTC(),
		Message: req.Message, Intent: req.Intent, SessionID: req.SessionID, History: req.History,
		User: dto.N8NUser{ID: u.ID, Name: u.Name},
		Context: dto.N8NChatContext{
			Date: today.Date, LocalTime: now.Format("15:04"), NextMealType: nextMealType(now.Hour()),
			Targets: today.Targets, Consumed: today.Consumed, Remaining: today.Remaining, Water: today.Water,
			Profile: dto.N8NProfile{
				DietPreference: prof.DietPreference, IsHalal: prof.IsHalal,
				Allergies: prof.Allergies, BudgetPreference: prof.BudgetPreference,
			},
			TodayLogs: briefs, CandidateFoods: sug.Items,
		},
	}

	var raw json.RawMessage
	if err := s.n8n.PostJSON(ctx, s.cfg.N8NChatURL, payload, &raw); err != nil {
		return nil, err
	}
	var reply dto.ChatReply
	if err := json.Unmarshal(unwrap(raw), &reply); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIBadResponse, err)
	}
	if strings.TrimSpace(reply.Reply) == "" {
		return nil, fmt.Errorf("%w: empty reply", ErrAIBadResponse)
	}
	if reply.Suggestions == nil {
		reply.Suggestions = []dto.Suggestion{}
	}
	return &reply, nil
}

// nextMealType: heuristik sederhana berdasarkan jam lokal.
func nextMealType(hour int) string {
	switch {
	case hour < 10:
		return "Breakfast"
	case hour < 14:
		return "Lunch"
	case hour < 21:
		return "Dinner"
	default:
		return "Snack"
	}
}

// unwrap: n8n sering membalas array [ {...} ]; ambil elemen pertama.
func unwrap(raw json.RawMessage) json.RawMessage {
	t := strings.TrimSpace(string(raw))
	if strings.HasPrefix(t, "[") {
		var arr []json.RawMessage
		if json.Unmarshal(raw, &arr) == nil && len(arr) > 0 {
			return arr[0]
		}
	}
	return raw
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
