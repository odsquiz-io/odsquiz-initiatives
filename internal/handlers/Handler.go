package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/services"
)

type Handler struct {
	Service *services.Service
}

func (h *Handler) CreateOne(c fiber.Ctx) error {
	one := new(models.Initiative)

	if err := c.Bind().Body(one); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(err.Error())
	}

	createdOne, err := h.Service.CreateOne(one)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(createdOne)
}

func (h *Handler) GetAllOnes(c fiber.Ctx) error {
	ones, err := h.Service.GetAllOnes()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(ones)
}

func (h *Handler) GetOneByID(c fiber.Ctx) error {
	id := c.Params("id")

	_, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(err.Error())
	}

	one, err := h.Service.GetOneByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(one)
}

func (h *Handler) UpdateOne(c fiber.Ctx) error {
	id := c.Params("id")

	_, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(err.Error())
	}

	one := new(models.Initiative)

	if err := c.Bind().Body(one); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(err.Error())
	}

	// force route param ID
	one.ID = id

	updatedOne, err := h.Service.UpdateOne(one)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(updatedOne)
}

func (h *Handler) DeleteOne(c fiber.Ctx) error {
	id := c.Params("id")

	_, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON("invalid id")
	}

	err = h.Service.DeleteOne(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err.Error())
	}

	return c.Status(fiber.StatusOK).JSON("deleted successfully")
}
