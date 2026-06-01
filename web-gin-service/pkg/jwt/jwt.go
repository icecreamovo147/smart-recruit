package jwt

import (
	"time"

	golangjwt "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     int32  `json:"role"`
	golangjwt.RegisteredClaims
}

const (
	// AccessTokenTTL is the short-lived access token TTL (24 hours).
	AccessTokenTTL = 24 * time.Hour
	// RefreshTokenTTL is the long-lived opaque refresh token TTL (30 days).
	RefreshTokenTTL = 30 * 24 * time.Hour
)

// Generate creates a new access token with the standard AccessTokenTTL.
func Generate(secret string, userID int64, username string, role int32) (string, error) {
	return GenerateWithTTL(secret, userID, username, role, AccessTokenTTL)
}

// GenerateWithTTL creates a new token with a custom TTL.
func GenerateWithTTL(secret string, userID int64, username string, role int32, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: golangjwt.RegisteredClaims{
			ExpiresAt: golangjwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  golangjwt.NewNumericDate(time.Now()),
		},
	}
	return golangjwt.NewWithClaims(golangjwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// Parse validates and parses a JWT token, returning the Claims if valid.
func Parse(tokenString string, claims *Claims, secret []byte) (*Claims, error) {
	token, err := golangjwt.ParseWithClaims(tokenString, claims, func(token *golangjwt.Token) (any, error) {
		if _, ok := token.Method.(*golangjwt.SigningMethodHMAC); !ok {
			return nil, golangjwt.ErrSignatureInvalid
		}
		return secret, nil
	}, golangjwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}
