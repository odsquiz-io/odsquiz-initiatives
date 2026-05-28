package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/migrations"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/routes"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	
	err = migrations.RunMigrations()
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New(fiber.Config{
		AppName: "ODS Quiz Initiatives Microservice",
	})

	app.Use(cors.New())

	routes.Setup(app)
	
	log.Fatal(app.Listen(":" + cfg.Port))
}
