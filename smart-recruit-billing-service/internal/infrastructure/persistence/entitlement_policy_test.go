package persistence

import (
	"context"
	"os"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"smart-recruit-billing-service/internal/domain/model"
)

func TestEntitlementBoolean(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "JSON boolean true", value: "true", want: true},
		{name: "JSON boolean true uppercase", value: " TRUE ", want: true},
		{name: "numeric true", value: "1", want: true},
		{name: "JSON boolean false", value: "false", want: false},
		{name: "numeric false", value: "0", want: false},
		{name: "invalid", value: "enabled", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := entitlementBoolean(tt.value); got != tt.want {
				t.Fatalf("entitlementBoolean(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestShanghaiMonthRetainsBusinessLocation(t *testing.T) {
	start, end := shanghaiMonth(time.Date(2026, 7, 31, 16, 30, 0, 0, time.UTC))
	if got := start.Format(time.RFC3339); got != "2026-08-01T00:00:00+08:00" {
		t.Fatalf("start = %s", got)
	}
	if got := end.Format(time.RFC3339); got != "2026-09-01T00:00:00+08:00" {
		t.Fatalf("end = %s", got)
	}
}

// This opt-in check targets a migrated, seeded MySQL database because SQLite
// cannot reproduce MySQL's JSON boolean conversion behavior.
func TestMySQLSeededCandidateFreeEntitlement(t *testing.T) {
	dsn := os.Getenv("BILLING_TEST_SEEDED_MYSQL_DSN")
	if dsn == "" {
		t.Skip("BILLING_TEST_SEEDED_MYSQL_DSN is not configured")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	policy, err := NewEntitlementPolicy(db)
	if err != nil {
		t.Fatal(err)
	}
	owner := model.Owner{Type: model.OwnerUser, ID: 987654321}
	enabled, err := policy.Enabled(context.Background(), owner, "ai.chat")
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("candidate_free ai.chat JSON boolean was not decoded as enabled")
	}
	releaseID, err := policy.ReleaseVersionID(context.Background(), owner, "ai.chat")
	if err != nil {
		t.Fatal(err)
	}
	var audience string
	if err := db.Raw(`SELECT capability.audience
		FROM platform_ai_capability_versions version
		JOIN platform_ai_capabilities capability ON capability.id = version.capability_id
		WHERE version.id = ?`, releaseID).Scan(&audience).Error; err != nil {
		t.Fatal(err)
	}
	if releaseID == 0 || audience != "candidate" {
		t.Fatalf("candidate_free release = %d (%s), want a candidate release", releaseID, audience)
	}
}
