package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestValidateCurrentPrincipalRejectsDeletedUser(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(42))
		c.Set("token_version", int32(1))
		c.Next()
	})
	router.Use(ValidateCurrentPrincipal(func(context.Context, int64) (*CurrentPrincipal, error) {
		return nil, nil
	}))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestValidateCurrentPrincipalRefreshesAuthorizationContext(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(7))
		c.Set("username", "stale")
		c.Set("role", int32(1))
		c.Set("account_type", "candidate")
		c.Set("roles", []string{"stale_role"})
		c.Set("permissions", []string{"stale_permission"})
		c.Set("token_version", int32(3))
		c.Next()
	})
	router.Use(ValidateCurrentPrincipal(func(_ context.Context, userID int64) (*CurrentPrincipal, error) {
		return &CurrentPrincipal{
			UserID:       userID,
			Username:     "current",
			Role:         2,
			AccountType:  "staff",
			Roles:        []string{"recruiter"},
			Permissions:  []string{"job:read"},
			TokenVersion: 3,
		}, nil
	}))
	router.GET("/protected", func(c *gin.Context) {
		if Username(c) != "current" || AccountType(c) != "staff" {
			t.Fatalf("principal was not refreshed: username=%q account_type=%q", Username(c), AccountType(c))
		}
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestValidateCurrentPrincipalRejectsTokenVersionMismatch(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(7))
		c.Set("token_version", int32(1))
		c.Next()
	})
	router.Use(ValidateCurrentPrincipal(func(_ context.Context, userID int64) (*CurrentPrincipal, error) {
		return &CurrentPrincipal{UserID: userID, TokenVersion: 2}, nil
	}))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
