package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/services"
)

type PostsHandler struct {
	Service *services.PostsService
}

func (h *PostsHandler) ListFeed(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	limit, err := postsLimitQuery(c)
	if err != nil {
		return respondError(c, err)
	}

	page, err := h.Service.ListFeed(userID, c.Query("cursor"), limit)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(page)
}

func (h *PostsHandler) ListInitiativePosts(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	initiativeID, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	limit, err := postsLimitQuery(c)
	if err != nil {
		return respondError(c, err)
	}

	page, err := h.Service.ListInitiativePosts(userID, initiativeID, c.Query("cursor"), limit)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(page)
}

func (h *PostsHandler) CreatePost(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	initiativeID, err := uuidParam(c, "id")
	if err != nil {
		return respondError(c, err)
	}

	request, err := postsBodyRequest(c)
	if err != nil {
		return respondError(c, err)
	}

	post, err := h.Service.CreatePost(userID, initiativeID, request.Body)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(post)
}

func (h *PostsHandler) UpdatePost(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	postID, err := uuidParam(c, "postId")
	if err != nil {
		return respondError(c, err)
	}

	request, err := postsBodyRequest(c)
	if err != nil {
		return respondError(c, err)
	}

	post, err := h.Service.UpdatePost(userID, postID, request.Body)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(post)
}

func (h *PostsHandler) DeletePost(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	postID, err := uuidParam(c, "postId")
	if err != nil {
		return respondError(c, err)
	}

	if err := h.Service.DeletePost(userID, postID); err != nil {
		return respondError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *PostsHandler) ListComments(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	postID, err := uuidParam(c, "postId")
	if err != nil {
		return respondError(c, err)
	}

	limit, err := postsLimitQuery(c)
	if err != nil {
		return respondError(c, err)
	}

	page, err := h.Service.ListComments(userID, postID, c.Query("cursor"), limit)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(page)
}

func (h *PostsHandler) CreateComment(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	postID, err := uuidParam(c, "postId")
	if err != nil {
		return respondError(c, err)
	}

	request, err := postsBodyRequest(c)
	if err != nil {
		return respondError(c, err)
	}

	comment, err := h.Service.CreateComment(userID, postID, request.Body)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(comment)
}

func (h *PostsHandler) UpdateComment(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	commentID, err := uuidParam(c, "commentId")
	if err != nil {
		return respondError(c, err)
	}

	request, err := postsBodyRequest(c)
	if err != nil {
		return respondError(c, err)
	}

	comment, err := h.Service.UpdateComment(userID, commentID, request.Body)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(comment)
}

func (h *PostsHandler) DeleteComment(c fiber.Ctx) error {
	userID, err := authenticatedUserID(c)
	if err != nil {
		return respondError(c, err)
	}

	commentID, err := uuidParam(c, "commentId")
	if err != nil {
		return respondError(c, err)
	}

	if err := h.Service.DeleteComment(userID, commentID); err != nil {
		return respondError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func postsLimitQuery(c fiber.Ctx) (int, error) {
	raw := c.Query("limit")
	if raw == "" {
		return 0, nil
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 {
		return 0, apperrors.BadRequest(apperrors.CodeInvalidRequest, err)
	}

	return limit, nil
}

func postsBodyRequest(c fiber.Ctx) (*models.PostBodyRequest, error) {
	request := new(models.PostBodyRequest)
	if err := c.Bind().Body(request); err != nil {
		return nil, apperrors.BadRequest(apperrors.CodeInvalidRequest, err)
	}

	return request, nil
}
