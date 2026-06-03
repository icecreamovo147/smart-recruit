package jwt

import (
	"time"

	golangjwt "github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT access token payload.
// During the migration window, both legacy Role and new RBAC fields are included.
type Claims struct {
	UserID       int64    `json:"user_id"`
	Username     string   `json:"username"`
	Role         int32    `json:"role"`          // Deprecated: kept for compatibility
	AccountType  string   `json:"account_type"`
	Roles        []string `json:"roles"`          // RBAC role keys
	Permissions  []string `json:"permissions"`    // RBAC permission keys
	TokenVersion int32    `json:"token_version"`  // Incremented on permission change
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

// GenerateWithTTL creates a new token with a custom TTL using legacy role only.
func GenerateWithTTL(secret string, userID int64, username string, role int32, ttl time.Duration) (string, error) {
	return GenerateFull(secret, userID, username, role, "", nil, nil, 1, ttl)
}

// GenerateFull creates a JWT with full RBAC metadata.
func GenerateFull(secret string, userID int64, username string, role int32, accountType string, roles, permissions []string, tokenVersion int32, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:       userID,
		Username:     username,
		Role:         role,
		AccountType:  accountType,
		Roles:        roles,
		Permissions:  permissions,
		TokenVersion: tokenVersion,
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
