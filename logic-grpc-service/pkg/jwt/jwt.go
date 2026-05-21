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

func Generate(secret string, userID int64, username string, role int32) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: golangjwt.RegisteredClaims{
			ExpiresAt: golangjwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  golangjwt.NewNumericDate(time.Now()),
		},
	}
	return golangjwt.NewWithClaims(golangjwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
