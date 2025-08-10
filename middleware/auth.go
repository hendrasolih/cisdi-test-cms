package middleware

import (
	"cisdi-test-cms/config"
	"cisdi-test-cms/helper"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

var HTTPHelper = &helper.HTTPHelper{}

var jwtKey = []byte(config.JWTSecret)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			HTTPHelper.SendUnauthorizedError(c, "Authorization header required", HTTPHelper.EmptyJsonMap())
			c.Abort()
			return
		}

		// Ambil token string
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			HTTPHelper.SendUnauthorizedError(c, "Bearer token required", HTTPHelper.EmptyJsonMap())
			c.Abort()
			return
		}
		fmt.Println("Token String:", tokenString)

		claims := &Claims{}

		// ✅ ParseWithClaims menggunakan pointer *jwt.Token
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Validasi metode signing
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtKey, nil
		})

		if err != nil {
			HTTPHelper.SendUnauthorizedError(c, "Invalid token: "+err.Error(), HTTPHelper.EmptyJsonMap())
			c.Abort()
			return
		}

		if !token.Valid {
			HTTPHelper.SendUnauthorizedError(c, "Token is not valid", HTTPHelper.EmptyJsonMap())
			c.Abort()
			return
		}

		// Simpan data ke context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			HTTPHelper.SendUnauthorizedError(c, "User role not found", HTTPHelper.EmptyJsonMap())
			c.Abort()
			return
		}

		roleStr := userRole.(string)
		for _, role := range roles {
			if roleStr == role {
				c.Next()
				return
			}
		}

		HTTPHelper.SendBadRequest(c, "Insufficient permissions", HTTPHelper.EmptyJsonMap())
		c.Abort()
	}
}
