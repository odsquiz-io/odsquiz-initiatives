package middleware

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kauanpecanha/odsquiz-initiatives/pkg/config"
)

// Protected function to setup Authorization Header handling
func Protected() fiber.Handler {

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	secretKey := []byte(cfg.JWTSecret)

	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		// missing authorization header error handling
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		// token extraction
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// missing "Bearer" error handling
		if tokenString == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization format",
			})
		}

		// token return
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			return secretKey, nil
		})

		// invalid/expired token error handling
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		return c.Next()
	}
}