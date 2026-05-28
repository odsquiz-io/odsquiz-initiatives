package migrations

import (
	"log"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/config"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/database"
)

func RunMigrations() error {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.AutoMigrate(&models.Initiative{}); err != nil {
		log.Fatal(err)
	}

	return err
}