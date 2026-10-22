package repositories

import (
	"time"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"gorm.io/gorm"
)

// users.id and initiatives.id are text when created by AutoMigrate and uuid when created by the goose
// migrations, so joins with them compare both sides as text.
const (
	postColumns = `p.id, p.initiative_id, p.author_id, p.body, p.created_at, p.edited_at, p.suspended_at,
		COALESCE(i.name, '') AS initiative_name, COALESCE(i.owner::text, '') AS initiative_owner,
		COALESCE(i.main_ods, 0) AS initiative_main_ods, COALESCE(i.city, '') AS initiative_city,
		COALESCE(i.state, '') AS initiative_state, i.logo_object AS initiative_logo_object,
		i.suspended_at AS initiative_suspended_at,
		COALESCE(u.name, '') AS author_name, u.suspended_at AS author_suspended_at,
		(SELECT COUNT(*) FROM initiative_post_comments AS c
			WHERE c.post_id = p.id AND c.suspended_at IS NULL AND EXISTS (
				SELECT 1 FROM users AS cu WHERE cu.id::text = c.author_id::text AND cu.suspended_at IS NULL
			)) AS comments_count`

	commentColumns = `c.id, c.post_id, c.author_id, c.body, c.created_at, c.edited_at, c.suspended_at,
		COALESCE(u.name, '') AS author_name, u.suspended_at AS author_suspended_at`

	visiblePost    = "p.suspended_at IS NULL AND i.suspended_at IS NULL AND u.suspended_at IS NULL"
	visibleComment = "c.suspended_at IS NULL AND u.suspended_at IS NULL"
)

type PostsRepository interface {
	ReadInitiative(id string) (*models.Initiative, error)
	IsActiveMember(initiativeID, userID string) (bool, error)
	ListPosts(filter models.PostFilter) ([]models.PostRecord, error)
	ReadPost(id string) (*models.PostRecord, error)
	CreatePost(post *models.InitiativePost) error
	UpdatePostBody(id, body string, editedAt time.Time) error
	DeletePost(id string) error
	ListComments(filter models.CommentFilter) ([]models.CommentRecord, error)
	ReadComment(id string) (*models.CommentRecord, error)
	CreateComment(comment *models.InitiativePostComment) error
	UpdateCommentBody(id, body string, editedAt time.Time) error
	DeleteComment(id string) error
}

type postsRepository struct {
	DB *gorm.DB
}

func NewPostsRepo(db *gorm.DB) PostsRepository {
	return &postsRepository{DB: db}
}

func (r *postsRepository) ReadInitiative(id string) (*models.Initiative, error) {
	var one models.Initiative

	err := r.DB.Select("id", "owner", "suspended_at").Take(&one, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &one, nil
}

func (r *postsRepository) IsActiveMember(initiativeID, userID string) (bool, error) {
	return IsActiveMember(r.DB, initiativeID, userID)
}

func (r *postsRepository) ListPosts(filter models.PostFilter) ([]models.PostRecord, error) {
	query := r.posts().Where(visiblePost)

	if filter.InitiativeID != "" {
		query = query.Where("p.initiative_id = ?", filter.InitiativeID)
	}
	if filter.Cursor != nil {
		query = query.Where("(p.created_at, p.id) < (?::timestamptz, ?::uuid)", filter.Cursor.CreatedAt, filter.Cursor.ID)
	}

	posts := []models.PostRecord{}
	err := query.Select(postColumns).Order("p.created_at DESC, p.id DESC").Limit(filter.Limit).Find(&posts).Error
	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *postsRepository) ReadPost(id string) (*models.PostRecord, error) {
	var post models.PostRecord

	err := r.posts().Select(postColumns).Where("p.id = ?", id).Take(&post).Error
	if err != nil {
		return nil, err
	}

	return &post, nil
}

func (r *postsRepository) CreatePost(post *models.InitiativePost) error {
	return r.DB.Create(post).Error
}

func (r *postsRepository) UpdatePostBody(id, body string, editedAt time.Time) error {
	return r.updateBody(&models.InitiativePost{}, id, body, editedAt)
}

// DeletePost removes the comments explicitly because tables created by AutoMigrate have no cascading keys.
func (r *postsRepository) DeletePost(id string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.InitiativePostComment{}, "post_id = ?", id).Error; err != nil {
			return err
		}

		result := tx.Delete(&models.InitiativePost{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

func (r *postsRepository) ListComments(filter models.CommentFilter) ([]models.CommentRecord, error) {
	query := r.comments().Where(visibleComment).Where("c.post_id = ?", filter.PostID)

	if filter.Cursor != nil {
		query = query.Where("(c.created_at, c.id) > (?::timestamptz, ?::uuid)", filter.Cursor.CreatedAt, filter.Cursor.ID)
	}

	comments := []models.CommentRecord{}
	err := query.Select(commentColumns).Order("c.created_at, c.id").Limit(filter.Limit).Find(&comments).Error
	if err != nil {
		return nil, err
	}

	return comments, nil
}

func (r *postsRepository) ReadComment(id string) (*models.CommentRecord, error) {
	var comment models.CommentRecord

	err := r.comments().Select(commentColumns).Where("c.id = ?", id).Take(&comment).Error
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

func (r *postsRepository) CreateComment(comment *models.InitiativePostComment) error {
	return r.DB.Create(comment).Error
}

func (r *postsRepository) UpdateCommentBody(id, body string, editedAt time.Time) error {
	return r.updateBody(&models.InitiativePostComment{}, id, body, editedAt)
}

func (r *postsRepository) DeleteComment(id string) error {
	result := r.DB.Delete(&models.InitiativePostComment{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *postsRepository) posts() *gorm.DB {
	return r.DB.Table("initiative_posts AS p").
		Joins("JOIN initiatives AS i ON i.id::text = p.initiative_id::text").
		Joins("JOIN users AS u ON u.id::text = p.author_id::text")
}

func (r *postsRepository) comments() *gorm.DB {
	return r.DB.Table("initiative_post_comments AS c").
		Joins("JOIN users AS u ON u.id::text = c.author_id::text")
}

func (r *postsRepository) updateBody(model any, id, body string, editedAt time.Time) error {
	result := r.DB.Model(model).
		Where("id = ?", id).
		Updates(map[string]any{"body": body, "edited_at": editedAt, "updated_at": editedAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
