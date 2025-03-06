package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// JWT returns a middleware that validates JWT tokens
func JWT(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Missing authorization header")
		}

		// Check if the header has the Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid authorization header format")
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token signing method")
			}
			return []byte(secret), nil
		})

		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid token: "+err.Error())
		}

		// Check if the token is valid
		if !token.Valid {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid token claims")
		}

		// Store claims in context for later use
		c.Locals("user", claims)

		return c.Next()
	}
}

// APIKey returns a middleware that validates API keys
func APIKey(validKeys []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get API key from header
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Missing API key")
		}

		// Check if the API key is valid
		for _, key := range validKeys {
			if apiKey == key {
				return c.Next()
			}
		}

		return fiber.NewError(fiber.StatusUnauthorized, "Invalid API key")
	}
}
