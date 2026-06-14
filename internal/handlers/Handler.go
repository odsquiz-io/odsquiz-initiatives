package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/services"
)

type Handler struct {
	Service *services.Service
}

func respondError(c fiber.Ctx, err error) error {
	appErr, ok := apperrors.From(err)
	if !ok {
		appErr = apperrors.Internal(err)
	}

	status := fiber.StatusInternalServerError
	switch appErr.Kind {
	case apperrors.KindBadRequest:
		status = fiber.StatusBadRequest
	case apperrors.KindUnauthorized:
		status = fiber.StatusUnauthorized
	case apperrors.KindNotFound:
		status = fiber.StatusNotFound
	case apperrors.KindConflict:
		status = fiber.StatusConflict
	case apperrors.KindInternal:
		status = fiber.StatusInternalServerError
	}

	return c.Status(status).JSON(fiber.Map{
		"code": appErr.Code,
	})
}

func (h *Handler) CreateOne(c fiber.Ctx) error {
	one := new(models.Initiative)

	if err := c.Bind().Body(one); err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	createdOne, err := h.Service.CreateOne(one)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(createdOne)
}

func (h *Handler) GetAllOnes(c fiber.Ctx) error {
	ones, err := h.Service.GetAllOnes()
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ones)
}

func (h *Handler) GetOneByID(c fiber.Ctx) error {
	id := c.Params("id")

	_, err := uuid.Parse(id)
	if err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	one, err := h.Service.GetOneByID(id)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(one)
}

func (h *Handler) UpdateOne(c fiber.Ctx) error {
	id := c.Params("id")

	_, err := uuid.Parse(id)
	if err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	one := new(models.Initiative)

	if err := c.Bind().Body(one); err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	// force route param ID
	one.ID = id

	updatedOne, err := h.Service.UpdateOne(one)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(updatedOne)
}

func (h *Handler) DeleteOne(c fiber.Ctx) error {
	id := c.Params("id")

	_, err := uuid.Parse(id)
	if err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	err = h.Service.DeleteOne(id)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON("deleted successfully")
}
