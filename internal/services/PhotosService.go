package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/repositories"
	"gorm.io/gorm"
)

const (
	MaxPhotos        = 12
	maxCaptionLength = 200
)

// ListPhotos returns the photos in order, the cover first; suspended initiatives show them only to their owners.
func (s *Service) ListPhotos(userID, id string) ([]models.Photo, error) {
	if _, err := s.readVisible(userID, id); err != nil {
		return nil, err
	}

	return s.listPhotos(id)
}

// AddPhoto stores the image and appends it after the initiative's last photo.
func (s *Service) AddPhoto(ctx context.Context, userID, id string, data []byte, caption *string) (*models.Photo, error) {
	caption, err := normalizeCaption(caption)
	if err != nil {
		return nil, err
	}

	if _, err := s.authorizeChange(userID, id); err != nil {
		return nil, err
	}

	contentType, extension, hash, err := inspectImage(data)
	if err != nil {
		return nil, err
	}

	photos, err := s.Photos.ListPhotos(id)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	if len(photos) >= MaxPhotos {
		return nil, apperrors.Conflict(apperrors.CodePhotoLimitReached, nil)
	}

	photo := &models.InitiativePhoto{
		ID:           uuid.NewString(),
		InitiativeID: id,
		ContentType:  contentType,
		Caption:      caption,
		CreatedAt:    time.Now(),
	}
	photo.Object = fmt.Sprintf("initiatives/%s/photos/%s-%s.%s", id, photo.ID, hash, extension)

	if err := s.Storage.Put(ctx, photo.Object, contentType, data); err != nil {
		return nil, mapStorageError(err, apperrors.CodePhotoNotFound)
	}

	if err := s.Photos.CreatePhoto(photo, MaxPhotos); err != nil {
		s.deleteObject(ctx, photo.Object)
		return nil, mapPhotoWriteError(err)
	}

	response := models.NewPhoto(*photo)
	return &response, nil
}

func (s *Service) UpdatePhotoCaption(userID, id, photoID string, caption *string) (*models.Photo, error) {
	caption, err := normalizeCaption(caption)
	if err != nil {
		return nil, err
	}

	if _, err := s.authorizeChange(userID, id); err != nil {
		return nil, err
	}

	if err := s.Photos.UpdateCaption(id, photoID, caption); err != nil {
		return nil, mapNotFound(err, apperrors.CodePhotoNotFound)
	}

	photo, err := s.Photos.ReadPhoto(id, photoID)
	if err != nil {
		return nil, mapNotFound(err, apperrors.CodePhotoNotFound)
	}

	response := models.NewPhoto(*photo)
	return &response, nil
}

// ReorderPhotos sets the order of the photos; photoIDs must list each current photo exactly once.
func (s *Service) ReorderPhotos(userID, id string, photoIDs []string) ([]models.Photo, error) {
	if _, err := s.authorizeChange(userID, id); err != nil {
		return nil, err
	}

	photos, err := s.Photos.ListPhotos(id)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	if !isPermutation(photos, photoIDs) {
		return nil, apperrors.BadRequest(apperrors.CodeInvalidRequest, nil)
	}

	if err := s.Photos.ReorderPhotos(id, photoIDs); err != nil {
		return nil, mapPhotoWriteError(err)
	}

	return s.listPhotos(id)
}

// DeletePhoto removes the photo, closing the gap it leaves in the order, and then its image.
func (s *Service) DeletePhoto(ctx context.Context, userID, id, photoID string) error {
	if _, err := s.authorizeChange(userID, id); err != nil {
		return err
	}

	photo, err := s.Photos.DeletePhoto(id, photoID)
	if err != nil {
		return mapNotFound(err, apperrors.CodePhotoNotFound)
	}

	s.deleteObject(ctx, photo.Object)
	return nil
}

// FindPhoto returns the image of a photo whose initiative is not suspended.
func (s *Service) FindPhoto(id, photoID string) (*Image, error) {
	one, err := s.Repo.ReadOneByID(id)
	if err != nil {
		return nil, mapNotFound(err, apperrors.CodePhotoNotFound)
	}

	if one.SuspendedAt != nil {
		return nil, apperrors.NotFound(apperrors.CodePhotoNotFound, nil)
	}

	photo, err := s.Photos.ReadPhoto(id, photoID)
	if err != nil {
		return nil, mapNotFound(err, apperrors.CodePhotoNotFound)
	}

	hash := photo.Hash()
	if hash == "" {
		return nil, apperrors.NotFound(apperrors.CodePhotoNotFound, nil)
	}

	return &Image{Object: photo.Object, ContentType: photo.ContentType, Hash: hash}, nil
}

func (s *Service) ReadPhoto(ctx context.Context, photo *Image) ([]byte, error) {
	return s.readImage(ctx, photo, apperrors.CodePhotoNotFound)
}

func (s *Service) listPhotos(id string) ([]models.Photo, error) {
	photos, err := s.Photos.ListPhotos(id)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	return models.NewPhotos(photos), nil
}

// normalizeCaption trims the caption; a blank caption is removed.
func normalizeCaption(caption *string) (*string, error) {
	if caption == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*caption)
	if trimmed == "" {
		return nil, nil
	}

	if utf8.RuneCountInString(trimmed) > maxCaptionLength {
		return nil, apperrors.BadRequest(apperrors.CodeInvalidRequest, nil)
	}

	return &trimmed, nil
}

func isPermutation(photos []models.InitiativePhoto, photoIDs []string) bool {
	if len(photos) != len(photoIDs) {
		return false
	}

	pending := make(map[string]bool, len(photos))
	for _, photo := range photos {
		pending[photo.ID] = true
	}

	for _, photoID := range photoIDs {
		if !pending[photoID] {
			return false
		}
		delete(pending, photoID)
	}

	return true
}

func mapPhotoWriteError(err error) error {
	switch {
	case errors.Is(err, repositories.ErrPhotoLimitReached):
		return apperrors.Conflict(apperrors.CodePhotoLimitReached, err)
	case errors.Is(err, repositories.ErrPhotosChanged):
		return apperrors.BadRequest(apperrors.CodeInvalidRequest, err)
	case errors.Is(err, gorm.ErrRecordNotFound):
		return apperrors.NotFound(apperrors.CodeInitiativeNotFound, err)
	}

	return apperrors.Internal(err)
}
