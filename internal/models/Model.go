package models

import (
	"time"
)

type Initiative struct {
	ID               string     `gorm:"primaryKey" json:"id"`
	Name             string     `json:"name"`
	Owner            string     `gorm:"type:uuid;column:owner;index:initiatives_owner_idx" json:"owner"`
	Address          string     `json:"address"`
	CEP              string     `json:"cep"`
	City             string     `gorm:"type:varchar(255)" json:"city"`
	State            string     `gorm:"type:varchar(2)" json:"state"`
	EmailOwner       string     `json:"email_owner"`
	ActingArea       string     `json:"acting_area"`
	Impact           int64      `json:"impact"`
	Type             string     `json:"type"`
	Points           int        `gorm:"default:0" json:"points"`
	MainODS          int        `json:"main_ods"`
	Lat              float64    `json:"lat"`
	Lon              float64    `json:"lon"`
	Description      *string    `gorm:"type:text" json:"-"`
	Whatsapp         *string    `gorm:"type:varchar(13)" json:"whatsapp"`
	Instagram        *string    `gorm:"type:varchar(30)" json:"instagram"`
	Website          *string    `gorm:"type:varchar(255)" json:"website"`
	LogoObject       *string    `gorm:"type:varchar(255)" json:"-"`
	LogoContentType  *string    `gorm:"type:varchar(32)" json:"-"`
	SuspendedAt      *time.Time `json:"suspended_at"`
	SuspensionReason *string    `gorm:"type:varchar(500)" json:"suspension_reason"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// InitiativeRequest holds the fields a client may set; ownership, points and moderation are server-side.
type InitiativeRequest struct {
	Name        string  `json:"name"`
	Address     string  `json:"address"`
	CEP         string  `json:"cep"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	EmailOwner  string  `json:"email_owner"`
	ActingArea  string  `json:"acting_area"`
	Impact      int64   `json:"impact"`
	Type        string  `json:"type"`
	MainODS     int     `json:"main_ods"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Description *string `json:"description"`
	Whatsapp    *string `json:"whatsapp"`
	Instagram   *string `json:"instagram"`
	Website     *string `json:"website"`
}

func (r *InitiativeRequest) Initiative() *Initiative {
	return &Initiative{
		Name:        r.Name,
		Address:     r.Address,
		CEP:         r.CEP,
		City:        r.City,
		State:       r.State,
		EmailOwner:  r.EmailOwner,
		ActingArea:  r.ActingArea,
		Impact:      r.Impact,
		Type:        r.Type,
		MainODS:     r.MainODS,
		Lat:         r.Lat,
		Lon:         r.Lon,
		Description: r.Description,
		Whatsapp:    r.Whatsapp,
		Instagram:   r.Instagram,
		Website:     r.Website,
	}
}

const (
	ViewerRoleOwner  = "owner"
	ViewerRoleMember = "member"
)

// InitiativeResponse is the initiative representation exposed over HTTP; logo storage details stay internal.
type InitiativeResponse struct {
	Initiative
	Description string  `json:"description"`
	LogoURL     *string `json:"logo_url"`
}

type InitiativeDetail struct {
	InitiativeResponse
	ViewerRole   *string `json:"viewer_role"`
	MembersCount int64   `json:"members_count"`
}

func NewInitiativeResponse(one Initiative) InitiativeResponse {
	response := InitiativeResponse{Initiative: one, LogoURL: LogoURL(one.ID, one.LogoObject)}
	if one.Description != nil {
		response.Description = *one.Description
	}

	return response
}

func NewInitiativeResponses(ones []Initiative) []InitiativeResponse {
	responses := make([]InitiativeResponse, 0, len(ones))
	for _, one := range ones {
		responses = append(responses, NewInitiativeResponse(one))
	}

	return responses
}

func (one Initiative) Summary() InitiativeSummary {
	return InitiativeSummary{
		ID:      one.ID,
		Name:    one.Name,
		MainODS: one.MainODS,
		City:    one.City,
		State:   one.State,
		LogoURL: LogoURL(one.ID, one.LogoObject),
	}
}
