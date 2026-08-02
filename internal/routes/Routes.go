package routes

import (
	"github.com/gofiber/fiber/v3"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/handlers"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/middleware"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/repositories"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/services"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/database"
)

func Setup(app *fiber.App) {
	repository := repositories.NewRepo(database.DB.Db)
	service := &services.Service{
		Repo: repository,
	}
	handler := &handlers.Handler{
		Service: service,
	}

	app.Get("/", func(c fiber.Ctx) error { return c.SendString("Hello, world!") })
	app.Get("/health", func(c fiber.Ctx) error { return c.SendString("ok") })
	// protected routes (can be used ONLY with bearer token)
	app.Post("/api/initiatives", middleware.Protected(), handler.CreateOne)
	app.Get("/api/initiatives", middleware.Protected(), handler.GetAllOnes)
	app.Get("/api/initiatives/:id", middleware.Protected(), handler.GetOneByID)
	app.Patch("/api/initiatives/:id", middleware.Protected(), handler.UpdateOne)
	app.Delete("/api/initiatives/:id", middleware.Protected(), handler.DeleteOne)
	app.Post("/createOne", middleware.Protected(), handler.CreateOne)
	app.Get("/getAllOnes", middleware.Protected(), handler.GetAllOnes)
	app.Get("/getOneById/:id", middleware.Protected(), handler.GetOneByID)
	app.Patch("/updateOne/:id", middleware.Protected(), handler.UpdateOne)
	app.Delete("/deleteOne/:id", middleware.Protected(), handler.DeleteOne)

	app.Use(func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNotFound) })
}
