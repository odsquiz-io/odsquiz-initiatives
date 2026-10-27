package models

import "time"

type InitiativePhoto struct {
	ID           string    `gorm:"primaryKey;type:uuid"`
	InitiativeID string    `gorm:"type:uuid;not null;index:initiative_photos_initiative_position_idx,priority:1"`
	Object       string    `gorm:"type:varchar(255);not null"`
	ContentType  string    `gorm:"type:varchar(32);not null"`
	Caption      *string   `gorm:"type:varchar(200)"`
	Position     int       `gorm:"type:integer;not null;index:initiative_photos_initiative_position_idx,priority:2"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`
}

func (InitiativePhoto) TableName() string {
	return "initiative_photos"
}

func PhotosModels() []any {
	return []any{&InitiativePhoto{}}
}

// Hash extracts the hash from an object named "initiatives/<id>/photos/<photoId>-<hash>.<ext>".
func (photo InitiativePhoto) Hash() string {
	return objectHash(photo.Object, photo.ID+"-")
}

// Photo is the photo representation exposed over HTTP; position 0 is the cover.
type Photo struct {
	ID       string  `json:"id"`
	URL      string  `json:"url"`
	Caption  *string `json:"caption"`
	Position int     `json:"position"`
}

func NewPhoto(photo InitiativePhoto) Photo {
	return Photo{
		ID:       photo.ID,
		URL:      "/api/initiatives/" + photo.InitiativeID + "/photos/" + photo.ID + "/image?v=" + photo.Hash(),
		Caption:  photo.Caption,
		Position: photo.Position,
	}
}

func NewPhotos(photos []InitiativePhoto) []Photo {
	responses := make([]Photo, 0, len(photos))
	for _, photo := range photos {
		responses = append(responses, NewPhoto(photo))
	}

	return responses
}

type PhotoCaptionRequest struct {
	Caption *string `json:"caption"`
}

type PhotoOrderRequest struct {
	PhotoIDs []string `json:"photo_ids"`
}
