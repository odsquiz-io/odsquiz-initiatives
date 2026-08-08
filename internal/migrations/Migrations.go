package migrations

import (
	"context"
	"fmt"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/config"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/database"
)

func RunMigrations(ctx context.Context, cfg *config.Config) error {
	db, err := database.NewPostgresConnection(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	if err := db.AutoMigrate(&models.Initiative{}); err != nil {
		return fmt.Errorf("migrate initiatives: %w", err)
	}

	return nil
}
