package routes

import (
	"github.com/gofiber/fiber/v3"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/handlers"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/middleware"
)

// RegisterMembers registers the member, invitation, membership and logo routes; it must run before
// /api/initiatives/:id, which would otherwise capture "invitations" and "memberships".
func RegisterMembers(app *fiber.App, handler *handlers.Handler) {
	protected := middleware.Protected()

	app.Get("/api/initiatives/invitations", protected, handler.ListInvitations)
	app.Post("/api/initiatives/invitations/:memberId/accept", protected, handler.AcceptInvitation)
	app.Post("/api/initiatives/invitations/:memberId/decline", protected, handler.DeclineInvitation)
	app.Get("/api/initiatives/memberships", protected, handler.ListMemberships)
	app.Get("/api/initiatives/:id/members", protected, handler.ListMembers)
	app.Post("/api/initiatives/:id/members", protected, handler.InviteMember)
	app.Delete("/api/initiatives/:id/members/:memberId", protected, handler.RemoveMember)
	app.Get("/api/initiatives/:id/logo", handler.GetLogo)
	app.Put("/api/initiatives/:id/logo", protected, handler.SetLogo)
	app.Delete("/api/initiatives/:id/logo", protected, handler.RemoveLogo)
}
