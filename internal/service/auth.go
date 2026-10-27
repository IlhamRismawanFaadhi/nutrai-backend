package service

import (
	"errors"
	"strings"

	"nutrai/internal/models"
	"nutrai/internal/pkg/apikey"
	"nutrai/internal/repository"
)

var ErrEmailTaken = errors.New("email already registered")

type AuthService struct{ repo *repository.Repository }

func NewAuthService(repo *repository.Repository) *AuthService { return &AuthService{repo: repo} }

// Register membuat user + profil default dan mengembalikan API key plaintext (sekali saja).
func (s *AuthService) Register(name, email string) (*models.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	exists, err := s.repo.EmailExists(email)
	if err != nil {
		return nil, "", err
	}
	if exists {
		return nil, "", ErrEmailTaken
	}
	plain, hash, err := apikey.Generate()
	if err != nil {
		return nil, "", err
	}
	u := &models.User{Name: strings.TrimSpace(name), Email: email, APIKey: hash}
	p := &models.UserProfile{
		CalorieTarget: 2000, ProteinTarget: 100, CarbsTarget: 250, FatTarget: 65,
		WaterTarget: 2.5, DietPreference: "none", BudgetPreference: "moderate",
		AllergiesJSON: "[]",
	}
	if err := s.repo.CreateUserWithProfile(u, p); err != nil {
		return nil, "", err
	}
	return u, plain, nil
}

// RotateKey mengganti API key lama dengan yang baru (key lama langsung tidak berlaku).
func (s *AuthService) RotateKey(userID uint) (string, error) {
	plain, hash, err := apikey.Generate()
	if err != nil {
		return "", err
	}
	return plain, s.repo.UpdateAPIKey(userID, hash)
}
