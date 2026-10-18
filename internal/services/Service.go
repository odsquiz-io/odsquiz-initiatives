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

func (s *Service) CreateOne(userID string, one *models.Initiative) (*models.Initiative, error) {
	if err := s.ensureActiveUser(userID); err != nil {
		return nil, err
	}

	one.Owner = userID

	createdOne, err := s.Repo.CreateOne(one)
	if err != nil {
		return nil, mapInitiativeWriteError(err)
	}

	return createdOne, nil
}

func (s *Service) GetActiveOnes() ([]models.Initiative, error) {
	ones, err := s.Repo.ReadActiveOnes()
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return ones, nil
}

func (s *Service) GetOnesByOwner(owner string) ([]models.Initiative, error) {
	ones, err := s.Repo.ReadOnesByOwner(owner)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return ones, nil
}

func (s *Service) GetOneByID(userID, id string) (*models.Initiative, error) {
	one, err := s.Repo.ReadOneByID(id)
	if err != nil {
		return nil, mapInitiativeReadError(err)
	}

	// Suspended initiatives stay visible only to their owners.
	if one.SuspendedAt != nil && one.Owner != userID {
		return nil, apperrors.NotFound(apperrors.CodeInitiativeNotFound, nil)
	}

	return one, nil
}

func (s *Service) UpdateOne(userID string, one *models.Initiative) (*models.Initiative, error) {
	if err := s.authorizeChange(userID, one.ID); err != nil {
		return nil, err
	}

	if _, err := s.Repo.UpdateOne(one); err != nil {
		return nil, mapInitiativeWriteError(err)
	}

	updatedOne, err := s.Repo.ReadOneByID(one.ID)
	if err != nil {
		return nil, mapInitiativeReadError(err)
	}

	return updatedOne, nil
}

func (s *Service) DeleteOne(userID, id string) error {
	if err := s.authorizeChange(userID, id); err != nil {
		return err
	}

	if err := s.Repo.DeleteOne(id); err != nil {
		return mapInitiativeReadError(err)
	}

	return nil
}

// authorizeChange allows changes only by the initiative's owner, while the account is not suspended.
func (s *Service) authorizeChange(userID, id string) error {
	if err := s.ensureActiveUser(userID); err != nil {
		return err
	}

	one, err := s.Repo.ReadOneByID(id)
	if err != nil {
		return mapInitiativeReadError(err)
	}

	if one.Owner != userID {
		return apperrors.Forbidden(apperrors.CodeForbidden, nil)
	}

	return nil
}

func (s *Service) ensureActiveUser(userID string) error {
	suspended, err := s.Repo.IsUserSuspended(userID)
	if err != nil {
		return apperrors.Internal(err)
	}

	if suspended {
		return apperrors.Forbidden(apperrors.CodeAccountSuspended, nil)
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
