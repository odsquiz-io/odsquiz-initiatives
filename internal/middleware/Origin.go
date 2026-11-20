package middleware

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v3"
)

// OriginHeader carries the secret the Cloudflare Worker adds to API requests.
const OriginHeader = "X-Origin-Token"

// RequireOrigin rejects requests that did not pass through the Cloudflare Worker.
// An empty token disables the check, as in the local Docker environment.
func RequireOrigin(token string) fiber.Handler {
	expected := []byte(token)

	return func(c fiber.Ctx) error {
		if token == "" || subtle.ConstantTimeCompare([]byte(c.Get(OriginHeader)), expected) == 1 {
			return c.Next()
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
}
