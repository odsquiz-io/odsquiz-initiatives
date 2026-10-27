package repositories

import (
	"errors"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrPhotoLimitReached = errors.New("photo limit reached")
	ErrPhotosChanged     = errors.New("photos changed")
)

// PhotosRepository keeps the positions of an initiative's photos contiguous from 0; writes that
// change positions lock the initiative row, so concurrent uploads and deletions are serialized.
type PhotosRepository interface {
	ListPhotos(initiativeID string) ([]models.InitiativePhoto, error)
	ReadPhoto(initiativeID, photoID string) (*models.InitiativePhoto, error)
	CreatePhoto(photo *models.InitiativePhoto, limit int) error
	UpdateCaption(initiativeID, photoID string, caption *string) error
	ReorderPhotos(initiativeID string, photoIDs []string) error
	DeletePhoto(initiativeID, photoID string) (*models.InitiativePhoto, error)
}

type photosRepository struct {
	DB *gorm.DB
}

func NewPhotosRepo(db *gorm.DB) PhotosRepository {
	return &photosRepository{DB: db}
}

func (r *photosRepository) ListPhotos(initiativeID string) ([]models.InitiativePhoto, error) {
	photos := []models.InitiativePhoto{}

	err := r.DB.Where("initiative_id = ?", initiativeID).Order("position, id").Find(&photos).Error
	if err != nil {
		return nil, err
	}

	return photos, nil
}

func (r *photosRepository) ReadPhoto(initiativeID, photoID string) (*models.InitiativePhoto, error) {
	var photo models.InitiativePhoto

	err := r.DB.Take(&photo, "id = ? AND initiative_id = ?", photoID, initiativeID).Error
	if err != nil {
		return nil, err
	}

	return &photo, nil
}

// CreatePhoto appends the photo after the initiative's last one, unless it already has limit photos.
func (r *photosRepository) CreatePhoto(photo *models.InitiativePhoto, limit int) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockInitiative(tx, photo.InitiativeID); err != nil {
			return err
		}

		var current struct {
			Count int
			Next  int
		}
		err := tx.Model(&models.InitiativePhoto{}).
			Select("COUNT(*) AS count, COALESCE(MAX(position) + 1, 0) AS next").
			Where("initiative_id = ?", photo.InitiativeID).
			Scan(&current).Error
		if err != nil {
			return err
		}

		if current.Count >= limit {
			return ErrPhotoLimitReached
		}

		photo.Position = current.Next
		return tx.Create(photo).Error
	})
}

func (r *photosRepository) UpdateCaption(initiativeID, photoID string, caption *string) error {
	result := r.DB.Model(&models.InitiativePhoto{}).
		Where("id = ? AND initiative_id = ?", photoID, initiativeID).
		Update("caption", caption)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// ReorderPhotos gives each photo its index in photoIDs, which must list every photo of the initiative.
func (r *photosRepository) ReorderPhotos(initiativeID string, photoIDs []string) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockInitiative(tx, initiativeID); err != nil {
			return err
		}

		var count int64
		if err := tx.Model(&models.InitiativePhoto{}).Where("initiative_id = ?", initiativeID).Count(&count).Error; err != nil {
			return err
		}

		if count != int64(len(photoIDs)) {
			return ErrPhotosChanged
		}

		for position, photoID := range photoIDs {
			result := tx.Model(&models.InitiativePhoto{}).
				Where("id = ? AND initiative_id = ?", photoID, initiativeID).
				Update("position", position)
			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected != 1 {
				return ErrPhotosChanged
			}
		}

		return nil
	})
}

// DeletePhoto removes the photo, moves the following ones back one position and returns the removed row.
func (r *photosRepository) DeletePhoto(initiativeID, photoID string) (*models.InitiativePhoto, error) {
	var photo models.InitiativePhoto

	err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := lockInitiative(tx, initiativeID); err != nil {
			return err
		}

		if err := tx.Take(&photo, "id = ? AND initiative_id = ?", photoID, initiativeID).Error; err != nil {
			return err
		}

		if err := tx.Delete(&models.InitiativePhoto{}, "id = ?", photoID).Error; err != nil {
			return err
		}

		return tx.Model(&models.InitiativePhoto{}).
			Where("initiative_id = ? AND position > ?", initiativeID, photo.Position).
			Update("position", gorm.Expr(`"position" - 1`)).Error
	})
	if err != nil {
		return nil, err
	}

	return &photo, nil
}

// lockInitiative locks the initiative row until the transaction ends; a missing initiative is not found.
func lockInitiative(tx *gorm.DB, id string) error {
	var one models.Initiative

	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Take(&one, "id = ?", id).Error
}
