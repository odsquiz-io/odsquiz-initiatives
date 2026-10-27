package services

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/repositories"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/storage"
	"gorm.io/gorm"
)

const maxDescriptionLength = 2000

type Service struct {
	Repo    repositories.Repository
	Members repositories.MembersRepository
	Logos   repositories.LogoRepository
	Photos  repositories.PhotosRepository
	Storage storage.Storage
}

func (s *Service) CreateOne(userID string, one *models.Initiative) (*models.Initiative, error) {
	if err := normalizeProfile(one); err != nil {
		return nil, err
	}

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

func (s *Service) GetOneByID(userID, id string) (*models.InitiativeDetail, error) {
	one, err := s.readVisible(userID, id)
	if err != nil {
		return nil, err
	}

	role, err := s.viewerRole(userID, one)
	if err != nil {
		return nil, err
	}

	count, err := s.Members.CountActiveMembers(id)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return &models.InitiativeDetail{
		InitiativeResponse: models.NewInitiativeResponse(*one),
		ViewerRole:         role,
		MembersCount:       count,
	}, nil
}

// readVisible reads an initiative; suspended initiatives stay visible only to their owners.
func (s *Service) readVisible(userID, id string) (*models.Initiative, error) {
	one, err := s.Repo.ReadOneByID(id)
	if err != nil {
		return nil, mapInitiativeReadError(err)
	}

	if one.SuspendedAt != nil && one.Owner != userID {
		return nil, apperrors.NotFound(apperrors.CodeInitiativeNotFound, nil)
	}

	return one, nil
}

func (s *Service) viewerRole(userID string, one *models.Initiative) (*string, error) {
	if one.Owner == userID {
		role := models.ViewerRoleOwner
		return &role, nil
	}

	member, err := s.Members.IsActiveMember(one.ID, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	if !member {
		return nil, nil
	}

	role := models.ViewerRoleMember
	return &role, nil
}

func (s *Service) UpdateOne(userID string, one *models.Initiative, columns ...string) (*models.Initiative, error) {
	if err := normalizeProfile(one); err != nil {
		return nil, err
	}

	if _, err := s.authorizeChange(userID, one.ID); err != nil {
		return nil, err
	}

	if _, err := s.Repo.UpdateOne(one, columns...); err != nil {
		return nil, mapInitiativeWriteError(err)
	}

	updatedOne, err := s.Repo.ReadOneByID(one.ID)
	if err != nil {
		return nil, mapInitiativeReadError(err)
	}

	return updatedOne, nil
}

func (s *Service) DeleteOne(userID, id string) error {
	one, err := s.authorizeChange(userID, id)
	if err != nil {
		return err
	}

	objects, err := s.Repo.DeleteOne(id)
	if err != nil {
		return mapInitiativeReadError(err)
	}

	if one.LogoObject != nil {
		objects = append(objects, *one.LogoObject)
	}

	if s.Storage != nil {
		for _, object := range objects {
			s.deleteObject(context.Background(), object)
		}
	}

	return nil
}

// authorizeChange allows changes only by the initiative's owner, while the account is not suspended.
func (s *Service) authorizeChange(userID, id string) (*models.Initiative, error) {
	if err := s.ensureActiveUser(userID); err != nil {
		return nil, err
	}

	one, err := s.Repo.ReadOneByID(id)
	if err != nil {
		return nil, mapInitiativeReadError(err)
	}

	if one.Owner != userID {
		return nil, apperrors.Forbidden(apperrors.CodeForbidden, nil)
	}

	return one, nil
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

func normalizeProfile(one *models.Initiative) error {
	if err := normalizeDescription(one); err != nil {
		return err
	}

	return normalizeContacts(one)
}

// normalizeDescription trims the "Sobre" text; a nil description is left unchanged.
func normalizeDescription(one *models.Initiative) error {
	if one.Description == nil {
		return nil
	}

	description := strings.TrimSpace(*one.Description)
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		return apperrors.BadRequest(apperrors.CodeInvalidRequest, nil)
	}

	one.Description = &description
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
