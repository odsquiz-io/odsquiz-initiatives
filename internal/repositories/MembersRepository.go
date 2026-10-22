package repositories

import (
	"time"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"gorm.io/gorm"
)

// users.id and initiatives.id are text when created by AutoMigrate and uuid when created by the goose
// migrations, so joins with initiative_members compare both sides as text.
const (
	memberColumns = `m.id, m.status, m.created_at, m.responded_at,
		u.id AS user_id, COALESCE(u.name, '') AS user_name, COALESCE(u.email, '') AS user_email`

	memberInitiativeColumns = `i.id AS initiative_id, COALESCE(i.name, '') AS initiative_name,
		COALESCE(i.main_ods, 0) AS main_ods, COALESCE(i.city, '') AS city, COALESCE(i.state, '') AS state,
		i.logo_object`
)

type MembersRepository interface {
	FindUser(id string) (*models.MemberAccount, error)
	FindUserByEmail(email string) (*models.MemberAccount, error)
	CreateMember(member *models.InitiativeMember) error
	ReadMember(id string) (*models.InitiativeMember, error)
	ReadMemberByUser(initiativeID, userID string) (*models.InitiativeMember, error)
	IsActiveMember(initiativeID, userID string) (bool, error)
	CountActiveMembers(initiativeID string) (int64, error)
	ListMembers(initiativeID string, withPending bool) ([]models.Member, error)
	ListInvitations(userID string) ([]models.Invitation, error)
	ListMemberships(userID string) ([]models.Membership, error)
	AcceptMember(id string, respondedAt time.Time) error
	DeleteMember(id string) error
}

type membersRepository struct {
	DB *gorm.DB
}

type memberRow struct {
	ID          string
	Status      string
	CreatedAt   time.Time
	RespondedAt *time.Time
	UserID      string
	UserName    string
	UserEmail   string
}

type memberInitiativeRow struct {
	ID             string
	CreatedAt      time.Time
	RespondedAt    *time.Time
	InitiativeID   string
	InitiativeName string
	MainODS        int
	City           string
	State          string
	LogoObject     *string
	InviterID      string
	InviterName    string
}

func NewMembersRepo(db *gorm.DB) MembersRepository {
	return &membersRepository{DB: db}
}

// IsActiveMember reports whether the user accepted an invitation to the initiative.
func IsActiveMember(db *gorm.DB, initiativeID, userID string) (bool, error) {
	var count int64

	err := db.Table("initiative_members").
		Where("initiative_id = ? AND user_id = ? AND status = ?", initiativeID, userID, models.MemberStatusActive).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *membersRepository) FindUser(id string) (*models.MemberAccount, error) {
	return r.findUser(r.DB.Where("id = ?", id))
}

func (r *membersRepository) FindUserByEmail(email string) (*models.MemberAccount, error) {
	return r.findUser(r.DB.Where("LOWER(email) = LOWER(?)", email).Order("created_at, id"))
}

func (r *membersRepository) findUser(query *gorm.DB) (*models.MemberAccount, error) {
	var user models.MemberAccount

	err := query.Table("users").
		Select("id, COALESCE(name, '') AS name, COALESCE(email, '') AS email, suspended_at").
		Take(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *membersRepository) CreateMember(member *models.InitiativeMember) error {
	return r.DB.Create(member).Error
}

func (r *membersRepository) ReadMember(id string) (*models.InitiativeMember, error) {
	var member models.InitiativeMember

	err := r.DB.Take(&member, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &member, nil
}

func (r *membersRepository) ReadMemberByUser(initiativeID, userID string) (*models.InitiativeMember, error) {
	var member models.InitiativeMember

	err := r.DB.Take(&member, "initiative_id = ? AND user_id = ?", initiativeID, userID).Error
	if err != nil {
		return nil, err
	}

	return &member, nil
}

func (r *membersRepository) IsActiveMember(initiativeID, userID string) (bool, error) {
	return IsActiveMember(r.DB, initiativeID, userID)
}

func (r *membersRepository) CountActiveMembers(initiativeID string) (int64, error) {
	var count int64

	err := r.DB.Model(&models.InitiativeMember{}).
		Where("initiative_id = ? AND status = ?", initiativeID, models.MemberStatusActive).
		Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *membersRepository) ListMembers(initiativeID string, withPending bool) ([]models.Member, error) {
	query := r.DB.Table("initiative_members AS m").
		Select(memberColumns).
		Joins("JOIN users AS u ON u.id::text = m.user_id::text").
		Where("m.initiative_id = ?", initiativeID)
	if !withPending {
		query = query.Where("m.status = ?", models.MemberStatusActive)
	}

	var rows []memberRow
	if err := query.Order("m.created_at, m.id").Scan(&rows).Error; err != nil {
		return nil, err
	}

	members := make([]models.Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, models.Member{
			ID:          row.ID,
			Status:      row.Status,
			User:        models.MemberUser{ID: row.UserID, Name: row.UserName, Email: row.UserEmail},
			CreatedAt:   row.CreatedAt,
			RespondedAt: row.RespondedAt,
		})
	}

	return members, nil
}

func (r *membersRepository) ListInvitations(userID string) ([]models.Invitation, error) {
	var rows []memberInitiativeRow

	err := r.memberInitiatives(userID, models.MemberStatusPending).
		Select("m.id, m.created_at, m.invited_by AS inviter_id, COALESCE(u.name, '') AS inviter_name, " + memberInitiativeColumns).
		Joins("LEFT JOIN users AS u ON u.id::text = m.invited_by::text").
		Order("m.created_at DESC, m.id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	invitations := make([]models.Invitation, 0, len(rows))
	for _, row := range rows {
		invitations = append(invitations, models.Invitation{
			ID:         row.ID,
			Initiative: row.summary(),
			InvitedBy:  models.MemberUser{ID: row.InviterID, Name: row.InviterName},
			CreatedAt:  row.CreatedAt,
		})
	}

	return invitations, nil
}

func (r *membersRepository) ListMemberships(userID string) ([]models.Membership, error) {
	var rows []memberInitiativeRow

	err := r.memberInitiatives(userID, models.MemberStatusActive).
		Select("m.id, m.responded_at, " + memberInitiativeColumns).
		Order("m.responded_at DESC, m.id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	memberships := make([]models.Membership, 0, len(rows))
	for _, row := range rows {
		memberships = append(memberships, models.Membership{
			MembershipID: row.ID,
			Initiative:   row.summary(),
			Since:        row.RespondedAt,
		})
	}

	return memberships, nil
}

// memberInitiatives selects the user's member rows whose initiative is not suspended.
func (r *membersRepository) memberInitiatives(userID, status string) *gorm.DB {
	return r.DB.Table("initiative_members AS m").
		Joins("JOIN initiatives AS i ON i.id::text = m.initiative_id::text").
		Where("m.user_id = ? AND m.status = ? AND i.suspended_at IS NULL", userID, status)
}

func (r *membersRepository) AcceptMember(id string, respondedAt time.Time) error {
	result := r.DB.Model(&models.InitiativeMember{}).
		Where("id = ? AND status = ?", id, models.MemberStatusPending).
		Updates(map[string]any{"status": models.MemberStatusActive, "responded_at": respondedAt})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *membersRepository) DeleteMember(id string) error {
	result := r.DB.Delete(&models.InitiativeMember{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (row memberInitiativeRow) summary() models.InitiativeSummary {
	return models.InitiativeSummary{
		ID:      row.InitiativeID,
		Name:    row.InitiativeName,
		MainODS: row.MainODS,
		City:    row.City,
		State:   row.State,
		LogoURL: models.LogoURL(row.InitiativeID, row.LogoObject),
	}
}
