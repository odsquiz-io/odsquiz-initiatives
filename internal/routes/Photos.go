package routes

import (
	"github.com/gofiber/fiber/v3"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/handlers"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/middleware"
)

// RegisterPhotos registers the photo routes; /photos/order stays before /photos/:photoId.
func RegisterPhotos(app *fiber.App, handler *handlers.Handler) {
	protected := middleware.Protected()

	app.Get("/api/initiatives/:id/photos", protected, handler.ListPhotos)
	app.Post("/api/initiatives/:id/photos", protected, handler.AddPhoto)
	app.Put("/api/initiatives/:id/photos/order", protected, handler.ReorderPhotos)
	app.Patch("/api/initiatives/:id/photos/:photoId", protected, handler.UpdatePhoto)
	app.Delete("/api/initiatives/:id/photos/:photoId", protected, handler.DeletePhoto)
	app.Get("/api/initiatives/:id/photos/:photoId/image", handler.GetPhotoImage)
}
