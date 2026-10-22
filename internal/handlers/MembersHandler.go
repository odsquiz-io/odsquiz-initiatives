package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
)

// uuidParam returns the route parameter as a canonical UUID.
func uuidParam(c fiber.Ctx, name string) (string, error) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil {
		return "", apperrors.BadRequest(apperrors.CodeInvalidRequest, err)
	}

	return id.String(), nil
}

func (h *Handler) ListMembers(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	members, err := h.Service.ListMembers(userID, id)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(members)
}

func (h *Handler) InviteMember(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	request := new(models.InviteRequest)
	if err := c.Bind().Body(request); err != nil {
		return respondError(c, apperrors.BadRequest(apperrors.CodeInvalidRequest, err))
	}

	member, err := h.Service.InviteMember(userID, id, request.Email)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(member)
}

func (h *Handler) RemoveMember(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	id, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	memberID, err := uuidParam(c, "memberId")
	if err != nil {
		return respondError(c, err)
	}

	if err := h.Service.RemoveMember(userID, id, memberID); err != nil {
		return respondError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) ListInvitations(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	invitations, err := h.Service.ListInvitations(userID)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(invitations)
}

func (h *Handler) AcceptInvitation(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	memberID, err := uuidParam(c, "memberId")
	if err != nil {
		return respondError(c, err)
	}

	accepted, err := h.Service.AcceptInvitation(userID, memberID)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(accepted)
}

func (h *Handler) DeclineInvitation(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	memberID, err := uuidParam(c, "memberId")
	if err != nil {
		return respondError(c, err)
	}

	if err := h.Service.DeclineInvitation(userID, memberID); err != nil {
		return respondError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) ListMemberships(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	memberships, err := h.Service.ListMemberships(userID)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(memberships)
}
