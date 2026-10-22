package models

import "time"

const (
	MemberStatusPending = "pending"
	MemberStatusActive  = "active"
)

// InitiativeMember is an invitation (pending) or a membership (active); the owner is never a member row.
type InitiativeMember struct {
	ID           string     `gorm:"primaryKey;type:uuid" json:"id"`
	InitiativeID string     `gorm:"type:uuid;not null;uniqueIndex:initiative_members_initiative_user_key" json:"initiative_id"`
	UserID       string     `gorm:"type:uuid;not null;uniqueIndex:initiative_members_initiative_user_key;index:initiative_members_user_status_idx" json:"user_id"`
	Status       string     `gorm:"type:varchar(16);not null;index:initiative_members_user_status_idx" json:"status"`
	InvitedBy    string     `gorm:"type:uuid;not null" json:"invited_by"`
	CreatedAt    time.Time  `gorm:"not null;default:now()" json:"created_at"`
	RespondedAt  *time.Time `json:"responded_at"`
}

func (InitiativeMember) TableName() string {
	return "initiative_members"
}

func MembersModels() []any {
	return []any{&InitiativeMember{}}
}

type MemberAccount struct {
	ID          string
	Name        string
	Email       string
	SuspendedAt *time.Time
}

type MemberUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type Member struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	User        MemberUser `json:"user"`
	CreatedAt   time.Time  `json:"created_at"`
	RespondedAt *time.Time `json:"responded_at"`
}

type MembersList struct {
	Owner   MemberUser `json:"owner"`
	Members []Member   `json:"members"`
}

type InviteRequest struct {
	Email string `json:"email"`
}

type Invitation struct {
	ID         string            `json:"id"`
	Initiative InitiativeSummary `json:"initiative"`
	InvitedBy  MemberUser        `json:"invited_by"`
	CreatedAt  time.Time         `json:"created_at"`
}

type AcceptedInvitation struct {
	ID         string            `json:"id"`
	Status     string            `json:"status"`
	Initiative InitiativeSummary `json:"initiative"`
}

type Membership struct {
	MembershipID string            `json:"membership_id"`
	Initiative   InitiativeSummary `json:"initiative"`
	Since        *time.Time        `json:"since"`
}
