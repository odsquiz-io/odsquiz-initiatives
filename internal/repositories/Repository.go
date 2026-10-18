package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"gorm.io/gorm"
)

type Repository interface {
	CreateOne(one *models.Initiative) (*models.Initiative, error)
	ReadActiveOnes() ([]models.Initiative, error)
	ReadOnesByOwner(owner string) ([]models.Initiative, error)
	ReadOneByID(id string) (*models.Initiative, error)
	ReadOneByEmail(email string) (*models.Initiative, error)
	UpdateOne(one *models.Initiative) (*models.Initiative, error)
	DeleteOne(id string) error
	IsUserSuspended(id string) (bool, error)
}

type repository struct {
	DB *gorm.DB
}

func NewRepo(db *gorm.DB) Repository {
	return &repository{
		DB: db,
	}
}

func (r *repository) CreateOne(one *models.Initiative) (*models.Initiative, error) {
	if one.ID == "" {
		one.ID = uuid.NewString()
	}

	one.CreatedAt = time.Now()
	one.UpdatedAt = time.Now()

	err := r.DB.Create(one).Error
	if err != nil {
		return nil, err
	}

	return one, nil
}

func (r *repository) ReadActiveOnes() ([]models.Initiative, error) {
	ones := []models.Initiative{}

	err := r.DB.Where("suspended_at IS NULL").Find(&ones).Error
	if err != nil {
		return nil, err
	}

	return ones, nil
}

func (r *repository) ReadOnesByOwner(owner string) ([]models.Initiative, error) {
	ones := []models.Initiative{}

	err := r.DB.Where("owner = ?", owner).Order("created_at DESC").Find(&ones).Error
	if err != nil {
		return nil, err
	}

	return ones, nil
}

func (r *repository) ReadOneByID(id string) (*models.Initiative, error) {
	var one models.Initiative

	err := r.DB.First(&one, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &one, nil
}

func (r *repository) ReadOneByEmail(email string) (*models.Initiative, error) {
	var one models.Initiative

	err := r.DB.First(&one, "email_owner = ?", email).Error
	if err != nil {
		return nil, err
	}

	return &one, nil
}

func (r *repository) UpdateOne(one *models.Initiative) (*models.Initiative, error) {
	one.UpdatedAt = time.Now()

	result := r.DB.Model(&models.Initiative{}).
		Where("id = ?", one.ID).
		Updates(one)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return one, nil
}

func (r *repository) DeleteOne(id string) error {
	result := r.DB.Delete(&models.Initiative{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *repository) IsUserSuspended(id string) (bool, error) {
	var count int64

	err := r.DB.Table("users").Where("id = ? AND suspended_at IS NOT NULL", id).Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
