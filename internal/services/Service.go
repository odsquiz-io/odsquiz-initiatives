package services

import (

	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/repositories"
)

type Service struct {
	Repo repositories.Repository
}

func (s *Service) CreateOne(one *models.Initiative) (*models.Initiative, error) {
	return s.Repo.CreateOne(one)
}

func (s *Service) GetAllOnes() ([]models.Initiative, error) {
	return s.Repo.ReadOnes()
}

func (s *Service) GetOneByID(id string) (*models.Initiative, error) {
	return s.Repo.ReadOneByID(id)
}

func (s *Service) UpdateOne(one *models.Initiative) (*models.Initiative, error) {
	return s.Repo.UpdateOne(one)
}

func (s *Service) DeleteOne(id string) error {
	return s.Repo.DeleteOne(id)
}
