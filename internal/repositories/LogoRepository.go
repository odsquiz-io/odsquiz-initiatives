package repositories

import (
	"time"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"gorm.io/gorm"
)

type LogoRepository interface {
	SetLogo(id string, object, contentType *string) error
}

type logoRepository struct {
	DB *gorm.DB
}

func NewLogoRepo(db *gorm.DB) LogoRepository {
	return &logoRepository{DB: db}
}

// SetLogo stores the logo object of an initiative; nil values clear it.
func (r *logoRepository) SetLogo(id string, object, contentType *string) error {
	result := r.DB.Model(&models.Initiative{}).
		Where("id = ?", id).
		Updates(map[string]any{"logo_object": object, "logo_content_type": contentType, "updated_at": time.Now()})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
