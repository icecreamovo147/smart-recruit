package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireActiveTenantFailsClosed(t *testing.T) {
	tests := []struct {
		name         string
		accountType  string
		tenantID     int64
		membershipID int64
		want         int
	}{
		{name: "active staff membership", accountType: "staff", tenantID: 7, membershipID: 9, want: http.StatusOK},
		{name: "staff without tenant", accountType: "staff", membershipID: 9, want: http.StatusForbidden},
		{name: "staff without membership", accountType: "staff", tenantID: 7, want: http.StatusForbidden},
		{name: "candidate cannot enter workspace", accountType: "candidate", tenantID: 7, membershipID: 9, want: http.StatusForbidden},
		{name: "platform cannot enter workspace", accountType: "platform", want: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := exerciseBoundary(t, func(c *gin.Context) {
				c.Set("account_type", test.accountType)
				c.Set("tenant_id", test.tenantID)
				c.Set("membership_id", test.membershipID)
			}, RequireActiveTenant())
			if recorder.Code != test.want {
				t.Fatalf("status = %d, want %d", recorder.Code, test.want)
			}
		})
	}
}

func TestRequirePlatformAppRejectsTenantAndWrongApplication(t *testing.T) {
	tests := []struct {
		name         string
		accountType  string
		clientApp    string
		tenantID     int64
		membershipID int64
		want         int
	}{
		{name: "platform principal", accountType: "platform", clientApp: "platform", want: http.StatusOK},
		{name: "platform role through HR app", accountType: "platform", clientApp: "hr", want: http.StatusForbidden},
		{name: "staff with platform header", accountType: "staff", clientApp: "platform", want: http.StatusForbidden},
		{name: "tenant-bound platform token", accountType: "platform", clientApp: "platform", tenantID: 7, membershipID: 9, want: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := exerciseBoundary(t, func(c *gin.Context) {
				c.Set("account_type", test.accountType)
				c.Set("client_app", test.clientApp)
				c.Set("tenant_id", test.tenantID)
				c.Set("membership_id", test.membershipID)
			}, RequirePlatformApp())
			if recorder.Code != test.want {
				t.Fatalf("status = %d, want %d", recorder.Code, test.want)
			}
		})
	}
}

func exerciseBoundary(t *testing.T, principal gin.HandlerFunc, boundary gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(principal, boundary)
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))
	return recorder
}
