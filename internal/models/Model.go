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
	SuspendedAt      *time.Time `json:"suspended_at"`
	SuspensionReason *string    `gorm:"type:varchar(500)" json:"suspension_reason"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// InitiativeRequest holds the fields a client may set; ownership, points and moderation are server-side.
type InitiativeRequest struct {
	Name       string  `json:"name"`
	Address    string  `json:"address"`
	CEP        string  `json:"cep"`
	City       string  `json:"city"`
	State      string  `json:"state"`
	EmailOwner string  `json:"email_owner"`
	ActingArea string  `json:"acting_area"`
	Impact     int64   `json:"impact"`
	Type       string  `json:"type"`
	MainODS    int     `json:"main_ods"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
}

func (r *InitiativeRequest) Initiative() *Initiative {
	return &Initiative{
		Name:       r.Name,
		Address:    r.Address,
		CEP:        r.CEP,
		City:       r.City,
		State:      r.State,
		EmailOwner: r.EmailOwner,
		ActingArea: r.ActingArea,
		Impact:     r.Impact,
		Type:       r.Type,
		MainODS:    r.MainODS,
		Lat:        r.Lat,
		Lon:        r.Lon,
	}
}
