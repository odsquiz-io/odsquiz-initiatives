package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/storage"
)

const MaxLogoSize = 2 << 20

var logoExtensions = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/webp": "webp",
}

type Logo struct {
	Object      string
	ContentType string
	Hash        string
}

// SetLogo stores a new logo, points the initiative to it and then removes the previous one.
func (s *Service) SetLogo(ctx context.Context, userID, id string, data []byte) (*models.Initiative, error) {
	one, err := s.authorizeChange(userID, id)
	if err != nil {
		return nil, err
	}

	if len(data) > MaxLogoSize {
		return nil, apperrors.BadRequest(apperrors.CodeImageTooLarge, nil)
	}

	contentType := http.DetectContentType(data)
	extension, ok := logoExtensions[contentType]
	if !ok {
		return nil, apperrors.BadRequest(apperrors.CodeInvalidImage, nil)
	}

	sum := sha256.Sum256(data)
	object := fmt.Sprintf("initiatives/%s/logo-%s.%s", id, hex.EncodeToString(sum[:])[:16], extension)
	previous := one.LogoObject

	if err := s.Storage.Put(ctx, object, contentType, data); err != nil {
		return nil, mapStorageError(err)
	}

	if err := s.Logos.SetLogo(id, &object, &contentType); err != nil {
		if previous == nil || *previous != object {
			s.deleteObject(ctx, object)
		}
		return nil, mapInitiativeWriteError(err)
	}

	if previous != nil && *previous != object {
		s.deleteObject(ctx, *previous)
	}

	return s.readOne(id)
}

func (s *Service) RemoveLogo(ctx context.Context, userID, id string) (*models.Initiative, error) {
	one, err := s.authorizeChange(userID, id)
	if err != nil {
		return nil, err
	}

	if one.LogoObject == nil {
		return one, nil
	}

	if err := s.Storage.Delete(ctx, *one.LogoObject); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, mapStorageError(err)
	}

	if err := s.Logos.SetLogo(id, nil, nil); err != nil {
		return nil, mapInitiativeWriteError(err)
	}

	return s.readOne(id)
}

// FindLogo returns the logo of an initiative that is not suspended.
func (s *Service) FindLogo(id string) (*Logo, error) {
	one, err := s.Repo.ReadOneByID(id)
	if err != nil {
		return nil, mapNotFound(err, apperrors.CodeLogoNotFound)
	}

	hash := models.LogoHash(one.LogoObject)
	if one.SuspendedAt != nil || hash == "" || one.LogoContentType == nil {
		return nil, apperrors.NotFound(apperrors.CodeLogoNotFound, nil)
	}

	return &Logo{Object: *one.LogoObject, ContentType: *one.LogoContentType, Hash: hash}, nil
}

func (s *Service) ReadLogo(ctx context.Context, logo *Logo) ([]byte, error) {
	data, err := s.Storage.Get(ctx, logo.Object)
	if err != nil {
		return nil, mapStorageError(err)
	}

	return data, nil
}

func (s *Service) readOne(id string) (*models.Initiative, error) {
	one, err := s.Repo.ReadOneByID(id)
	if err != nil {
		return nil, mapInitiativeReadError(err)
	}

	return one, nil
}

// deleteObject removes an object that is no longer referenced; a failure only leaves an orphan behind.
func (s *Service) deleteObject(ctx context.Context, object string) {
	if err := s.Storage.Delete(ctx, object); err != nil && !errors.Is(err, storage.ErrNotFound) {
		log.Printf("delete logo object %s: %v", object, err)
	}
}

func mapStorageError(err error) error {
	if errors.Is(err, storage.ErrNotFound) {
		return apperrors.NotFound(apperrors.CodeLogoNotFound, err)
	}

	return apperrors.Unavailable(apperrors.CodeStorageUnavailable, err)
}
