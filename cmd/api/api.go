package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/migrations"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/routes"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/config"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), database.StartupConnectionTimeout)
	defer cancel()

	err = migrations.RunMigrations(startupCtx, cfg)
	if err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}

	app := fiber.New(fiber.Config{
		AppName: "ODS Quiz Initiatives Microservice",
	})

	app.Use(cors.New())

	routes.Setup(app)

	log.Fatal(app.Listen(":" + cfg.Port))
}
