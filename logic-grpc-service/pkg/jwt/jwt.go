package jwt

import (
	"crypto/rand"
	"encoding/hex"
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
	// AccessTokenTTL is the time-to-live for short-lived access tokens.
	AccessTokenTTL = 24 * time.Hour
	// RefreshTokenTTL is the time-to-live for long-lived refresh tokens (30 days).
	RefreshTokenTTL = 30 * 24 * time.Hour
)

func Generate(secret string, userID int64, username string, role int32) (string, error) {
	return GenerateWithTTL(secret, userID, username, role, AccessTokenTTL)
}

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

// Parse validates and parses a JWT token. Returns the Claims if valid, or an error.
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

// GenerateRefreshToken generates a cryptographically random opaque refresh token.
// It is NOT a JWT — just a random hex string used to look up the associated user.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
