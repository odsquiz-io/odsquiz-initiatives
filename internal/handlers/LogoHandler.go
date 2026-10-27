package handlers

import (
	"context"
	"io"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/services"
)

const storageTimeout = 10 * time.Second

func (h *Handler) SetLogo(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	data, err := formImage(c, "logo")
	if err != nil {
		return respondError(c, err)
	}

	ctx, cancel := context.WithTimeout(c.Context(), storageTimeout)
	defer cancel()

	one, err := h.Service.SetLogo(ctx, userID, id, data)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(models.NewInitiativeResponse(*one))
}

func (h *Handler) RemoveLogo(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	ctx, cancel := context.WithTimeout(c.Context(), storageTimeout)
	defer cancel()

	one, err := h.Service.RemoveLogo(ctx, userID, id)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(models.NewInitiativeResponse(*one))
}

// GetLogo is public so that images load without the bearer token.
func (h *Handler) GetLogo(c fiber.Ctx) error {
	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	logo, err := h.Service.FindLogo(id)
	if err != nil {
		return respondError(c, err)
	}

	return sendImage(c, logo, h.Service.ReadLogo)
}

// formImage reads an uploaded image from a multipart field, refusing files above services.MaxImageSize.
func formImage(c fiber.Ctx, field string) ([]byte, error) {
	file, err := c.FormFile(field)
	if err != nil {
		return nil, apperrors.BadRequest(apperrors.CodeInvalidRequest, err)
	}

	if file.Size > services.MaxImageSize {
		return nil, apperrors.BadRequest(apperrors.CodeImageTooLarge, nil)
	}

	content, err := file.Open()
	if err != nil {
		return nil, apperrors.BadRequest(apperrors.CodeInvalidRequest, err)
	}
	defer func() { _ = content.Close() }()

	data, err := io.ReadAll(io.LimitReader(content, services.MaxImageSize+1))
	if err != nil {
		return nil, apperrors.BadRequest(apperrors.CodeInvalidRequest, err)
	}

	return data, nil
}

// sendImage answers with a stored image, cached for good when the URL carries its current hash.
func sendImage(c fiber.Ctx, image *services.Image, read func(context.Context, *services.Image) ([]byte, error)) error {
	etag := `"` + image.Hash + `"`
	cacheControl := "public, max-age=300"
	if c.Query("v") == image.Hash {
		cacheControl = "public, max-age=31536000, immutable"
	}

	if c.Get(fiber.HeaderIfNoneMatch) == etag {
		c.Set(fiber.HeaderETag, etag)
		c.Set(fiber.HeaderCacheControl, cacheControl)
		return c.SendStatus(fiber.StatusNotModified)
	}

	ctx, cancel := context.WithTimeout(c.Context(), storageTimeout)
	defer cancel()

	data, err := read(ctx, image)
	if err != nil {
		return respondError(c, err)
	}

	c.Set(fiber.HeaderContentType, image.ContentType)
	c.Set(fiber.HeaderETag, etag)
	c.Set(fiber.HeaderCacheControl, cacheControl)
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	return c.Status(fiber.StatusOK).Send(data)
}
