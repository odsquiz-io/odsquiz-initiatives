package models

import (
	"time"
)

type Initiative struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	Name           string    `json:"name"`
	Owner          string    `json:"owner"`
	Address        string    `json:"address"`
	CEP            string    `json:"cep"`
	EmailOwner     string    `json:"email_owner"`
	ActingArea     string    `json:"acting_area"`
	Impact				 int64     `json:"impact"`
	Type           string    `json:"type"`
	Points         int       `gorm:"default:0" json:"points"`
	MainODS        int       `json:"main_ods"`
	Lat            float64   `json:"lat"`
	Lon            float64   `json:"lon"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}