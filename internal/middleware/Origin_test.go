package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRequireOrigin(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		header string
		status int
	}{
		{name: "disabled without token", status: fiber.StatusOK},
		{name: "missing header", token: "secret", status: fiber.StatusForbidden},
		{name: "wrong header", token: "secret", header: "other", status: fiber.StatusForbidden},
		{name: "matching header", token: "secret", header: "secret", status: fiber.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(RequireOrigin(test.token))
			app.Get("/", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.header != "" {
				request.Header.Set(OriginHeader, test.header)
			}

			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != test.status {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.status)
			}
		})
	}
}
