package services

import (
	"errors"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/repositories"
	"gorm.io/gorm"
)

type Service struct {
	Repo repositories.Repository
}

func (s *Service) CreateOne(one *models.Initiative) (*models.Initiative, error) {
	createdOne, err := s.Repo.CreateOne(one)
	if err != nil {
		return nil, mapInitiativeWriteError(err)
	}

	return createdOne, nil
}

func (s *Service) GetAllOnes() ([]models.Initiative, error) {
	ones, err := s.Repo.ReadOnes()
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return ones, nil
}

func (s *Service) GetOneByID(id string) (*models.Initiative, error) {
	one, err := s.Repo.ReadOneByID(id)
	if err != nil {
		return nil, mapInitiativeReadError(err)
	}

	return one, nil
}

func (s *Service) UpdateOne(one *models.Initiative) (*models.Initiative, error) {
	updatedOne, err := s.Repo.UpdateOne(one)
	if err != nil {
		return nil, mapInitiativeWriteError(err)
	}

	return updatedOne, nil
}

func (s *Service) DeleteOne(id string) error {
	if err := s.Repo.DeleteOne(id); err != nil {
		return mapInitiativeReadError(err)
	}

	return nil
}

func mapInitiativeReadError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.NotFound(apperrors.CodeInitiativeNotFound, err)
	}

	return apperrors.Internal(err)
}

func mapInitiativeWriteError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.NotFound(apperrors.CodeInitiativeNotFound, err)
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperrors.Conflict(apperrors.CodeEmailAlreadyExists, err)
	}

	return apperrors.Internal(err)
}
