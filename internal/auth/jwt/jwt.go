package jwt

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"volumetric-backend/internal/auth/models"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET")) 

// GenerateAccessToken returns signed JWT
func GenerateAccessToken(user *models.User, roles, permissions []string) (string, error) {
	// Claims = Data stored inside token.
	claims := models.Claims{
		UserID:      user.UserID,
		Email:       user.Email,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "volumetric-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateAccessToken 
func ValidateAccessToken(tokenString string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*models.Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrSignatureInvalid
}

// (6) This file handles JWT token generation and validation.

// JWT = JSON Web Token

// Used for authentication (login system)