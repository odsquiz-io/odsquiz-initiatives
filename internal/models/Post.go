package models

import "time"

type InitiativePost struct {
	ID               string    `gorm:"primaryKey;type:uuid;index:initiative_posts_initiative_created_idx,priority:3,sort:desc;index:initiative_posts_created_idx,priority:2,sort:desc"`
	InitiativeID     string    `gorm:"type:uuid;not null;index:initiative_posts_initiative_created_idx,priority:1"`
	AuthorID         string    `gorm:"type:uuid;not null"`
	Body             string    `gorm:"type:text;not null"`
	CreatedAt        time.Time `gorm:"not null;default:now();index:initiative_posts_initiative_created_idx,priority:2,sort:desc;index:initiative_posts_created_idx,priority:1,sort:desc"`
	UpdatedAt        time.Time `gorm:"not null;default:now()"`
	EditedAt         *time.Time
	SuspendedAt      *time.Time
	SuspensionReason *string `gorm:"type:varchar(500)"`
}

func (InitiativePost) TableName() string {
	return "initiative_posts"
}

type InitiativePostComment struct {
	ID               string    `gorm:"primaryKey;type:uuid;index:initiative_post_comments_post_created_idx,priority:3"`
	PostID           string    `gorm:"type:uuid;not null;index:initiative_post_comments_post_created_idx,priority:1"`
	AuthorID         string    `gorm:"type:uuid;not null"`
	Body             string    `gorm:"type:text;not null"`
	CreatedAt        time.Time `gorm:"not null;default:now();index:initiative_post_comments_post_created_idx,priority:2"`
	UpdatedAt        time.Time `gorm:"not null;default:now()"`
	EditedAt         *time.Time
	SuspendedAt      *time.Time
	SuspensionReason *string `gorm:"type:varchar(500)"`
}

func (InitiativePostComment) TableName() string {
	return "initiative_post_comments"
}

func PostsModels() []any {
	return []any{&InitiativePost{}, &InitiativePostComment{}}
}

type FeedCursor struct {
	CreatedAt time.Time
	ID        string
}

type PostFilter struct {
	InitiativeID string
	Cursor       *FeedCursor
	Limit        int
}

type CommentFilter struct {
	PostID string
	Cursor *FeedCursor
	Limit  int
}

// PostRecord is a post joined with its initiative and author, as read from the database.
type PostRecord struct {
	ID                    string
	InitiativeID          string
	AuthorID              string
	Body                  string
	CreatedAt             time.Time
	EditedAt              *time.Time
	SuspendedAt           *time.Time
	InitiativeName        string
	InitiativeOwner       string
	InitiativeMainODS     int
	InitiativeCity        string
	InitiativeState       string
	InitiativeLogoObject  *string
	InitiativeSuspendedAt *time.Time
	AuthorName            string
	AuthorSuspendedAt     *time.Time
	CommentsCount         int64
}

type CommentRecord struct {
	ID                string
	PostID            string
	AuthorID          string
	Body              string
	CreatedAt         time.Time
	EditedAt          *time.Time
	SuspendedAt       *time.Time
	AuthorName        string
	AuthorSuspendedAt *time.Time
}

type PostAuthor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PostResponse struct {
	ID              string            `json:"id"`
	Initiative      InitiativeSummary `json:"initiative"`
	Author          PostAuthor        `json:"author"`
	Body            string            `json:"body"`
	CreatedAt       time.Time         `json:"created_at"`
	EditedAt        *time.Time        `json:"edited_at"`
	CommentsCount   int64             `json:"comments_count"`
	ViewerCanEdit   bool              `json:"viewer_can_edit"`
	ViewerCanDelete bool              `json:"viewer_can_delete"`
}

type CommentResponse struct {
	ID              string     `json:"id"`
	PostID          string     `json:"post_id"`
	Author          PostAuthor `json:"author"`
	Body            string     `json:"body"`
	CreatedAt       time.Time  `json:"created_at"`
	EditedAt        *time.Time `json:"edited_at"`
	ViewerCanEdit   bool       `json:"viewer_can_edit"`
	ViewerCanDelete bool       `json:"viewer_can_delete"`
}

type PostPage struct {
	Items      []PostResponse `json:"items"`
	NextCursor *string        `json:"next_cursor"`
}

type CommentPage struct {
	Items      []CommentResponse `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

type PostBodyRequest struct {
	Body string `json:"body"`
}
