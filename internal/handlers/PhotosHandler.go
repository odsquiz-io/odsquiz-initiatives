package handlers

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
)

func (h *Handler) ListPhotos(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	photos, err := h.Service.ListPhotos(userID, id)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(photos)
}

func (h *Handler) AddPhoto(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	data, err := formImage(c, "photo")
	if err != nil {
		return respondError(c, err)
	}

	caption := c.FormValue("caption")

	ctx, cancel := context.WithTimeout(c.Context(), storageTimeout)
	defer cancel()

	photo, err := h.Service.AddPhoto(ctx, userID, id, data, &caption)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(photo)
}

func (h *Handler) UpdatePhoto(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	photoID, err := uuidParam(c, "photoId")
	if err != nil {
		return respondError(c, err)
	}

	request := new(models.PhotoCaptionRequest)
	if err := c.Bind().Body(request); err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	photo, err := h.Service.UpdatePhotoCaption(userID, id, photoID, request.Caption)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(photo)
}

func (h *Handler) ReorderPhotos(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	request := new(models.PhotoOrderRequest)
	if err := c.Bind().Body(request); err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	photoIDs := make([]string, 0, len(request.PhotoIDs))
	for _, photoID := range request.PhotoIDs {
		parsed, err := uuid.Parse(photoID)
		if err != nil {
			return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
		}
		photoIDs = append(photoIDs, parsed.String())
	}

	photos, err := h.Service.ReorderPhotos(userID, id, photoIDs)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(photos)
}

func (h *Handler) DeletePhoto(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	photoID, err := uuidParam(c, "photoId")
	if err != nil {
		return respondError(c, err)
	}

	ctx, cancel := context.WithTimeout(c.Context(), storageTimeout)
	defer cancel()

	if err := h.Service.DeletePhoto(ctx, userID, id, photoID); err != nil {
		return respondError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetPhotoImage is public so that images load without the bearer token.
func (h *Handler) GetPhotoImage(c fiber.Ctx) error {
	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	photoID, err := uuidParam(c, "photoId")
	if err != nil {
		return respondError(c, err)
	}

	photo, err := h.Service.FindPhoto(id, photoID)
	if err != nil {
		return respondError(c, err)
	}

	return sendImage(c, photo, h.Service.ReadPhoto)
}
