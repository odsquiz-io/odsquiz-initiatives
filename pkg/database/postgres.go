// pkg/database/postgres.go: provides functions for postgres database connections and operations.
package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/cenkalti/backoff/v5"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kauanpecanha/odsquiz-initiatives/pkg/config"
)

type Dbinstance struct {
	Db *gorm.DB
}

var DB Dbinstance

const (
	maxConnectionAttempts       = 7
	initialConnectionRetryDelay = time.Second
	maxConnectionRetryDelay     = 15 * time.Second
	pingTimeout                 = 5 * time.Second

	// StartupConnectionTimeout bounds database initialization well below Cloud
	// Run's container startup deadline.
	StartupConnectionTimeout = 2 * time.Minute
)

// NewPostgresConnection establishes a connection to the PostgreSQL database using GORM.
func NewPostgresConnection(ctx context.Context, cfg *config.Config) (*gorm.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=5",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	db, err := backoff.Retry(ctx, func() (*gorm.DB, error) {
		db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
			Logger:         logger.Default.LogMode(logger.Info),
			TranslateError: true,
		})
		if err == nil {
			sqlDB, sqlErr := db.DB()
			if sqlErr != nil {
				err = sqlErr
			} else {
				pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
				err = sqlDB.PingContext(pingCtx)
				cancel()
				if err != nil {
					_ = sqlDB.Close()
				}
			}
		}

		return db, err
	},
		backoff.WithBackOff(newDatabaseBackOff()),
		backoff.WithMaxTries(maxConnectionAttempts),
		backoff.WithMaxElapsedTime(StartupConnectionTimeout),
		backoff.WithNotify(func(_ error, nextBackOff time.Duration) {
			// Deliberately avoid logging the connection error because it can include
			// details from the DSN, including credentials.
			log.Printf("database is unavailable; retrying in %s", nextBackOff)
		}),
	)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("database startup cancelled: %w", ctx.Err())
		}
		return nil, errors.New("database did not become available before the startup retry limit")
	}

	DB = Dbinstance{Db: db}
	return db, nil
}

func newDatabaseBackOff() *backoff.ExponentialBackOff {
	retryBackOff := backoff.NewExponentialBackOff()
	retryBackOff.InitialInterval = initialConnectionRetryDelay
	retryBackOff.RandomizationFactor = 0.2
	retryBackOff.Multiplier = 2
	retryBackOff.MaxInterval = maxConnectionRetryDelay
	return retryBackOff
}

// AutoMigrate allows automigration action at any part of the code
func AutoMigrate(db *gorm.DB, models ...any) error {
	return db.AutoMigrate(models...)
}
