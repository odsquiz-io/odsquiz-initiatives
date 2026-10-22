package services

import (
	"encoding/base64"
	"errors"
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
	maxPostBodyLength    = 3000
	maxCommentBodyLength = 1000

	defaultPostsLimit    = 20
	maxPostsLimit        = 50
	defaultCommentsLimit = 50
	maxCommentsLimit     = 100
)

type PostsService struct {
	Repo        repositories.PostsRepository
	Initiatives *Service
}

func (s *PostsService) ListFeed(viewerID, cursor string, limit int) (*models.PostPage, error) {
	after, err := decodeFeedCursor(cursor)
	if err != nil {
		return nil, err
	}

	return s.listPosts(viewerID, models.PostFilter{Cursor: after, Limit: limit})
}

// ListInitiativePosts hides suspended initiatives from everyone but their owners, who get an empty list.
func (s *PostsService) ListInitiativePosts(viewerID, initiativeID, cursor string, limit int) (*models.PostPage, error) {
	after, err := decodeFeedCursor(cursor)
	if err != nil {
		return nil, err
	}

	one, err := s.Repo.ReadInitiative(initiativeID)
	if err != nil {
		return nil, mapInitiativeReadError(err)
	}

	if one.SuspendedAt != nil {
		if one.Owner != viewerID {
			return nil, apperrors.NotFound(apperrors.CodeInitiativeNotFound, nil)
		}

		return &models.PostPage{Items: []models.PostResponse{}}, nil
	}

	return s.listPosts(viewerID, models.PostFilter{InitiativeID: initiativeID, Cursor: after, Limit: limit})
}

func (s *PostsService) CreatePost(userID, initiativeID, body string) (*models.PostResponse, error) {
	body, err := validBody(body, maxPostBodyLength)
	if err != nil {
		return nil, err
	}

	if err := s.Initiatives.ensureActiveUser(userID); err != nil {
		return nil, err
	}

	if err := s.authorizePublishing(userID, initiativeID); err != nil {
		return nil, err
	}

	now := feedNow()
	post := &models.InitiativePost{
		ID:           uuid.NewString(),
		InitiativeID: initiativeID,
		AuthorID:     userID,
		Body:         body,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.Repo.CreatePost(post); err != nil {
		return nil, apperrors.Internal(err)
	}

	record, err := s.Repo.ReadPost(post.ID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	response := postResponse(userID, record)
	return &response, nil
}

func (s *PostsService) UpdatePost(userID, postID, body string) (*models.PostResponse, error) {
	body, err := validBody(body, maxPostBodyLength)
	if err != nil {
		return nil, err
	}

	record, err := s.visiblePostForWrite(userID, postID)
	if err != nil {
		return nil, err
	}

	if record.AuthorID != userID {
		return nil, apperrors.Forbidden(apperrors.CodeForbidden, nil)
	}

	if body != record.Body {
		now := feedNow()
		if err := s.Repo.UpdatePostBody(postID, body, now); err != nil {
			return nil, mapPostError(err)
		}

		record.Body = body
		record.EditedAt = &now
	}

	response := postResponse(userID, record)
	return &response, nil
}

func (s *PostsService) DeletePost(userID, postID string) error {
	record, err := s.visiblePostForWrite(userID, postID)
	if err != nil {
		return err
	}

	if record.AuthorID != userID && record.InitiativeOwner != userID {
		return apperrors.Forbidden(apperrors.CodeForbidden, nil)
	}

	if err := s.Repo.DeletePost(postID); err != nil {
		return mapPostError(err)
	}

	return nil
}

func (s *PostsService) ListComments(viewerID, postID, cursor string, limit int) (*models.CommentPage, error) {
	after, err := decodeFeedCursor(cursor)
	if err != nil {
		return nil, err
	}

	post, err := s.visiblePost(postID)
	if err != nil {
		return nil, err
	}

	limit = pageLimit(limit, defaultCommentsLimit, maxCommentsLimit)
	records, err := s.Repo.ListComments(models.CommentFilter{PostID: postID, Cursor: after, Limit: limit + 1})
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	records, next := trimPage(records, limit, func(record models.CommentRecord) (time.Time, string) {
		return record.CreatedAt, record.ID
	})

	page := &models.CommentPage{Items: make([]models.CommentResponse, 0, len(records)), NextCursor: next}
	for index := range records {
		page.Items = append(page.Items, commentResponse(viewerID, post.InitiativeOwner, &records[index]))
	}

	return page, nil
}

func (s *PostsService) CreateComment(userID, postID, body string) (*models.CommentResponse, error) {
	body, err := validBody(body, maxCommentBodyLength)
	if err != nil {
		return nil, err
	}

	if err := s.Initiatives.ensureActiveUser(userID); err != nil {
		return nil, err
	}

	post, err := s.visiblePost(postID)
	if err != nil {
		return nil, err
	}

	now := feedNow()
	comment := &models.InitiativePostComment{
		ID:        uuid.NewString(),
		PostID:    postID,
		AuthorID:  userID,
		Body:      body,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.Repo.CreateComment(comment); err != nil {
		return nil, apperrors.Internal(err)
	}

	record, err := s.Repo.ReadComment(comment.ID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	response := commentResponse(userID, post.InitiativeOwner, record)
	return &response, nil
}

func (s *PostsService) UpdateComment(userID, commentID, body string) (*models.CommentResponse, error) {
	body, err := validBody(body, maxCommentBodyLength)
	if err != nil {
		return nil, err
	}

	record, post, err := s.visibleCommentForWrite(userID, commentID)
	if err != nil {
		return nil, err
	}

	if record.AuthorID != userID {
		return nil, apperrors.Forbidden(apperrors.CodeForbidden, nil)
	}

	if body != record.Body {
		now := feedNow()
		if err := s.Repo.UpdateCommentBody(commentID, body, now); err != nil {
			return nil, mapCommentError(err)
		}

		record.Body = body
		record.EditedAt = &now
	}

	response := commentResponse(userID, post.InitiativeOwner, record)
	return &response, nil
}

func (s *PostsService) DeleteComment(userID, commentID string) error {
	record, post, err := s.visibleCommentForWrite(userID, commentID)
	if err != nil {
		return err
	}

	if record.AuthorID != userID && post.InitiativeOwner != userID {
		return apperrors.Forbidden(apperrors.CodeForbidden, nil)
	}

	if err := s.Repo.DeleteComment(commentID); err != nil {
		return mapCommentError(err)
	}

	return nil
}

func (s *PostsService) listPosts(viewerID string, filter models.PostFilter) (*models.PostPage, error) {
	limit := pageLimit(filter.Limit, defaultPostsLimit, maxPostsLimit)
	filter.Limit = limit + 1

	records, err := s.Repo.ListPosts(filter)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	records, next := trimPage(records, limit, func(record models.PostRecord) (time.Time, string) {
		return record.CreatedAt, record.ID
	})

	page := &models.PostPage{Items: make([]models.PostResponse, 0, len(records)), NextCursor: next}
	for index := range records {
		page.Items = append(page.Items, postResponse(viewerID, &records[index]))
	}

	return page, nil
}

// authorizePublishing lets the owner or an active member publish while the initiative is not suspended.
func (s *PostsService) authorizePublishing(userID, initiativeID string) error {
	one, err := s.Repo.ReadInitiative(initiativeID)
	if err != nil {
		return mapInitiativeReadError(err)
	}

	if one.Owner == userID {
		if one.SuspendedAt != nil {
			return apperrors.Forbidden(apperrors.CodeForbidden, nil)
		}

		return nil
	}

	if one.SuspendedAt != nil {
		return apperrors.NotFound(apperrors.CodeInitiativeNotFound, nil)
	}

	member, err := s.Repo.IsActiveMember(initiativeID, userID)
	if err != nil {
		return apperrors.Internal(err)
	}

	if !member {
		return apperrors.Forbidden(apperrors.CodeForbidden, nil)
	}

	return nil
}

func (s *PostsService) visiblePost(postID string) (*models.PostRecord, error) {
	record, err := s.Repo.ReadPost(postID)
	if err != nil {
		return nil, mapPostError(err)
	}

	if !postVisible(record) {
		return nil, apperrors.NotFound(apperrors.CodePostNotFound, nil)
	}

	return record, nil
}

func (s *PostsService) visiblePostForWrite(userID, postID string) (*models.PostRecord, error) {
	if err := s.Initiatives.ensureActiveUser(userID); err != nil {
		return nil, err
	}

	return s.visiblePost(postID)
}

// visibleCommentForWrite returns the comment with its post, both visible, for a non-suspended user.
func (s *PostsService) visibleCommentForWrite(userID, commentID string) (*models.CommentRecord, *models.PostRecord, error) {
	if err := s.Initiatives.ensureActiveUser(userID); err != nil {
		return nil, nil, err
	}

	record, err := s.Repo.ReadComment(commentID)
	if err != nil {
		return nil, nil, mapCommentError(err)
	}

	if !commentVisible(record) {
		return nil, nil, apperrors.NotFound(apperrors.CodeCommentNotFound, nil)
	}

	post, err := s.Repo.ReadPost(record.PostID)
	if err != nil {
		return nil, nil, mapCommentError(err)
	}

	if !postVisible(post) {
		return nil, nil, apperrors.NotFound(apperrors.CodeCommentNotFound, nil)
	}

	return record, post, nil
}

func postVisible(record *models.PostRecord) bool {
	return record.SuspendedAt == nil && record.InitiativeSuspendedAt == nil && record.AuthorSuspendedAt == nil
}

func commentVisible(record *models.CommentRecord) bool {
	return record.SuspendedAt == nil && record.AuthorSuspendedAt == nil
}

func postResponse(viewerID string, record *models.PostRecord) models.PostResponse {
	return models.PostResponse{
		ID: record.ID,
		Initiative: models.InitiativeSummary{
			ID:      record.InitiativeID,
			Name:    record.InitiativeName,
			MainODS: record.InitiativeMainODS,
			City:    record.InitiativeCity,
			State:   record.InitiativeState,
			LogoURL: models.LogoURL(record.InitiativeID, record.InitiativeLogoObject),
		},
		Author:          models.PostAuthor{ID: record.AuthorID, Name: record.AuthorName},
		Body:            record.Body,
		CreatedAt:       record.CreatedAt,
		EditedAt:        record.EditedAt,
		CommentsCount:   record.CommentsCount,
		ViewerCanEdit:   viewerID == record.AuthorID,
		ViewerCanDelete: viewerID == record.AuthorID || viewerID == record.InitiativeOwner,
	}
}

func commentResponse(viewerID, initiativeOwner string, record *models.CommentRecord) models.CommentResponse {
	return models.CommentResponse{
		ID:              record.ID,
		PostID:          record.PostID,
		Author:          models.PostAuthor{ID: record.AuthorID, Name: record.AuthorName},
		Body:            record.Body,
		CreatedAt:       record.CreatedAt,
		EditedAt:        record.EditedAt,
		ViewerCanEdit:   viewerID == record.AuthorID,
		ViewerCanDelete: viewerID == record.AuthorID || viewerID == initiativeOwner,
	}
}

func validBody(body string, maxLength int) (string, error) {
	body = strings.TrimSpace(body)

	length := utf8.RuneCountInString(body)
	if length == 0 || length > maxLength || !utf8.ValidString(body) || strings.ContainsRune(body, 0) {
		return "", apperrors.BadRequest(apperrors.CodeInvalidRequest, nil)
	}

	return body, nil
}

func pageLimit(limit, defaultLimit, maxLimit int) int {
	if limit <= 0 {
		return defaultLimit
	}

	return min(limit, maxLimit)
}

// trimPage cuts a page fetched with one extra row and returns the cursor of the last kept row when more remain.
func trimPage[T any](records []T, limit int, key func(T) (time.Time, string)) ([]T, *string) {
	if len(records) <= limit {
		return records, nil
	}

	records = records[:limit]
	cursor := encodeFeedCursor(key(records[limit-1]))
	return records, &cursor
}

func encodeFeedCursor(createdAt time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(createdAt.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeFeedCursor(cursor string) (*models.FeedCursor, error) {
	if cursor == "" {
		return nil, nil
	}

	invalid := apperrors.BadRequest(apperrors.CodeInvalidRequest, nil)

	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(cursor, "="))
	if err != nil {
		return nil, invalid
	}

	createdAtText, idText, found := strings.Cut(string(raw), "|")
	if !found {
		return nil, invalid
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtText)
	if err != nil {
		return nil, invalid
	}

	id, err := uuid.Parse(idText)
	if err != nil {
		return nil, invalid
	}

	return &models.FeedCursor{CreatedAt: createdAt, ID: id.String()}, nil
}

// feedNow matches the microsecond precision of PostgreSQL timestamps, so responses equal what is stored.
func feedNow() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

func mapPostError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.NotFound(apperrors.CodePostNotFound, err)
	}

	return apperrors.Internal(err)
}

func mapCommentError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.NotFound(apperrors.CodeCommentNotFound, err)
	}

	return apperrors.Internal(err)
}
