package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/middleware"
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
	case apperrors.KindForbidden:
		status = fiber.StatusForbidden
	case apperrors.KindNotFound:
		status = fiber.StatusNotFound
	case apperrors.KindConflict:
		status = fiber.StatusConflict
	case apperrors.KindUnavailable:
		status = fiber.StatusServiceUnavailable
	case apperrors.KindInternal:
		status = fiber.StatusInternalServerError
	}

	return c.Status(status).JSON(fiber.Map{
		"code": appErr.Code,
	})
}

func authenticatedUserID(c fiber.Ctx) (string, error) {
	userID, _ := c.Locals(middleware.ContextUserID).(string)
	if userID == "" {
		return "", apperrors.Unauthorized(apperrors.CodeInvalidCredentials, nil)
	}

	return userID, nil
}

func (h *Handler) CreateOne(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	request := new(models.InitiativeRequest)

	if err := c.Bind().Body(request); err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	createdOne, err := h.Service.CreateOne(userID, request.Initiative())
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(models.NewInitiativeResponse(*createdOne))
}

func (h *Handler) GetAllOnes(c fiber.Ctx) error {
	ones, err := h.Service.GetActiveOnes()
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(models.NewInitiativeResponses(ones))
}

func (h *Handler) GetMine(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	ones, err := h.Service.GetOnesByOwner(userID)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(models.NewInitiativeResponses(ones))
}

func (h *Handler) GetOneByID(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id := c.Params("id")

	_, err = uuid.Parse(id)
	if err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	one, err := h.Service.GetOneByID(userID, id)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(one)
}

func (h *Handler) UpdateOne(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id := c.Params("id")

	_, err = uuid.Parse(id)
	if err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	request := new(models.InitiativeRequest)

	if err := c.Bind().Body(request); err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	columns, err := sentColumns(c.Body())
	if err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	one := request.Initiative()
	one.ID = id

	updatedOne, err := h.Service.UpdateOne(userID, one, columns...)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(models.NewInitiativeResponse(*updatedOne))
}

func (h *Handler) DeleteOne(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id := c.Params("id")

	_, err = uuid.Parse(id)
	if err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	err = h.Service.DeleteOne(userID, id)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON("deleted successfully")
}

// updatableColumns lists the request keys a PATCH may change; their JSON names match the columns.
var updatableColumns = []string{
	"name", "address", "cep", "city", "state", "email_owner", "acting_area",
	"impact", "type", "main_ods", "lat", "lon", "description", "whatsapp", "instagram", "website",
}

// sentColumns returns the updatable columns present in the body, so a PATCH can set zero values.
func sentColumns(body []byte) ([]string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, err
	}

	columns := make([]string, 0, len(fields))
	for _, column := range updatableColumns {
		if _, ok := fields[column]; ok {
			columns = append(columns, column)
		}
	}

	return columns, nil
}
