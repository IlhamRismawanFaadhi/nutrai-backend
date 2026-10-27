package service

import (
	"nutrai/internal/dto"
	"nutrai/internal/models"
	"nutrai/internal/repository"
)

type ProfileService struct{ repo *repository.Repository }

func NewProfileService(repo *repository.Repository) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) Get(u *models.User) (*dto.ProfileResponse, error) {
	p, err := s.repo.GetProfile(u.ID)
	if err != nil {
		return nil, err
	}
	return toProfileDTO(u, p), nil
}

func (s *ProfileService) Update(u *models.User, req dto.UpdateProfileRequest) (*dto.ProfileResponse, error) {
	p, err := s.repo.GetProfile(u.ID)
	if err != nil {
		return nil, err
	}
	if req.CalorieTarget != nil {
		p.CalorieTarget = *req.CalorieTarget
	}
	if req.ProteinTarget != nil {
		p.ProteinTarget = *req.ProteinTarget
	}
	if req.CarbsTarget != nil {
		p.CarbsTarget = *req.CarbsTarget
	}
	if req.FatTarget != nil {
		p.FatTarget = *req.FatTarget
	}
	if req.WaterTargetL != nil {
		p.WaterTarget = *req.WaterTargetL
	}
	if req.DietPreference != nil {
		p.DietPreference = *req.DietPreference
	}
	if req.IsHalal != nil {
		p.IsHalal = *req.IsHalal
	}
	if req.Allergies != nil {
		p.AllergiesJSON = tagsToJSON(*req.Allergies)
	}
	if req.BudgetPreference != nil {
		p.BudgetPreference = *req.BudgetPreference
	}
	if err := s.repo.SaveProfile(p); err != nil {
		return nil, err
	}
	return toProfileDTO(u, p), nil
}

func toProfileDTO(u *models.User, p *models.UserProfile) *dto.ProfileResponse {
	return &dto.ProfileResponse{
		Name: u.Name, Email: u.Email,
		CalorieTarget: p.CalorieTarget, ProteinTarget: p.ProteinTarget,
		CarbsTarget: p.CarbsTarget, FatTarget: p.FatTarget, WaterTargetL: p.WaterTarget,
		DietPreference: p.DietPreference, IsHalal: p.IsHalal,
		Allergies: jsonToTags(p.AllergiesJSON), BudgetPreference: p.BudgetPreference,
	}
}
