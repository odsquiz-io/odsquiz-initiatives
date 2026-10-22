package routes

import (
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/handlers"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/middleware"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/repositories"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/services"
)

// RegisterPosts must run before the /api/initiatives/:id routes, which would otherwise capture "feed".
func RegisterPosts(app *fiber.App, db *gorm.DB, initiatives *services.Service) {
	handler := &handlers.PostsHandler{
		Service: &services.PostsService{
			Repo:        repositories.NewPostsRepo(db),
			Initiatives: initiatives,
		},
	}
	protected := middleware.Protected()

	app.Get("/api/initiatives/feed", protected, handler.ListFeed)
	app.Patch("/api/initiatives/posts/:postId", protected, handler.UpdatePost)
	app.Delete("/api/initiatives/posts/:postId", protected, handler.DeletePost)
	app.Get("/api/initiatives/posts/:postId/comments", protected, handler.ListComments)
	app.Post("/api/initiatives/posts/:postId/comments", protected, handler.CreateComment)
	app.Patch("/api/initiatives/comments/:commentId", protected, handler.UpdateComment)
	app.Delete("/api/initiatives/comments/:commentId", protected, handler.DeleteComment)
	app.Get("/api/initiatives/:id/posts", protected, handler.ListInitiativePosts)
	app.Post("/api/initiatives/:id/posts", protected, handler.CreatePost)
}
