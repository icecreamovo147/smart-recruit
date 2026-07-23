package quota

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCheckerEnforcesHardQuota(t *testing.T) {
	db := quotaFixture(t, "hard", "2")
	if err := db.Exec("INSERT INTO jobs (tenant_id, status) VALUES (7, 1), (7, 1)").Error; err != nil {
		t.Fatal(err)
	}
	checker := NewChecker(db)
	result, err := checker.Check(context.Background(), 7, "jobs.published.max", 1)
	if !errors.Is(err, ErrLimitExceeded) || result.Allowed || result.Usage != 2 || result.Limit != 2 {
		t.Fatalf("unexpected quota result: result=%+v err=%v", result, err)
	}
}

func TestCheckerAllowsSoftQuotaAndTenantWithoutSubscription(t *testing.T) {
	db := quotaFixture(t, "soft", "1")
	if err := db.Exec("INSERT INTO jobs (tenant_id, status) VALUES (7, 1)").Error; err != nil {
		t.Fatal(err)
	}
	checker := NewChecker(db)
	if result, err := checker.Check(context.Background(), 7, "jobs.published.max", 1); err != nil || !result.Allowed {
		t.Fatalf("soft quota should allow: result=%+v err=%v", result, err)
	}
	if result, err := checker.Check(context.Background(), 99, "jobs.published.max", 1); err != nil || !result.Allowed {
		t.Fatalf("tenant without subscription should remain backward compatible: result=%+v err=%v", result, err)
	}
}

func TestCheckerUsesActiveTenantOverride(t *testing.T) {
	db := quotaFixture(t, "hard", "2")
	if err := db.Exec("INSERT INTO jobs (tenant_id, status) VALUES (7, 1), (7, 1)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO tenant_entitlement_overrides (tenant_id, entitlement_key, value_json, expires_at) VALUES (7, 'jobs.published.max', '4', ?)", time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	result, err := NewChecker(db).Check(context.Background(), 7, "jobs.published.max", 1)
	if err != nil || !result.Allowed || result.Limit != 4 {
		t.Fatalf("active override should replace plan quota: result=%+v err=%v", result, err)
	}
}

func TestCheckerCountsAllSupportedMetricsWithCalendarMonth(t *testing.T) {
	db := quotaFixture(t, "hard", "100")
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	statements := []string{
		"INSERT INTO tenant_memberships (tenant_id, status) VALUES (7, 'active'), (7, 'suspended'), (7, 'active')",
		"INSERT INTO jobs (tenant_id, status) VALUES (7, 1), (7, 0), (7, 1)",
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec(
		"INSERT INTO applications (tenant_id, resume_id, applied_at) VALUES (7, 11, ?), (7, 11, ?), (7, 12, ?)",
		time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.June, 30, 23, 59, 59, 0, time.UTC),
	).Error; err != nil {
		t.Fatal(err)
	}
	checker := NewChecker(db)
	checker.now = func() time.Time { return now }
	tests := []struct {
		key  string
		want int64
	}{
		{key: "members.max", want: 2},
		{key: "jobs.published.max", want: 2},
		{key: "applications.monthly.max", want: 2},
		{key: "resumes.storage.max", want: 2},
	}
	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			got, err := checker.currentUsage(context.Background(), 7, test.key)
			if err != nil || got != test.want {
				t.Fatalf("currentUsage() = %d, %v; want %d", got, err, test.want)
			}
		})
	}
}

func quotaFixture(t *testing.T, mode, value string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	ddl := []string{
		"CREATE TABLE tenant_subscriptions (id INTEGER PRIMARY KEY, tenant_id INTEGER, plan_version_id INTEGER, status TEXT, starts_at DATETIME, ends_at DATETIME)",
		"CREATE TABLE platform_plan_versions (id INTEGER PRIMARY KEY)",
		"CREATE TABLE platform_plan_entitlements (plan_version_id INTEGER, entitlement_key TEXT, value_json TEXT, enforcement_mode TEXT)",
		"CREATE TABLE tenant_entitlement_overrides (tenant_id INTEGER, entitlement_key TEXT, value_json TEXT, expires_at DATETIME)",
		"CREATE TABLE tenant_memberships (tenant_id INTEGER, status TEXT)",
		"CREATE TABLE jobs (tenant_id INTEGER, status INTEGER)",
		"CREATE TABLE applications (tenant_id INTEGER, resume_id INTEGER, applied_at DATETIME)",
	}
	for _, statement := range ddl {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec("INSERT INTO platform_plan_versions (id) VALUES (3)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO tenant_subscriptions (id, tenant_id, plan_version_id, status, starts_at) VALUES (1, 7, 3, 'active', ?)", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO platform_plan_entitlements (plan_version_id, entitlement_key, value_json, enforcement_mode) VALUES (3, 'jobs.published.max', ?, ?)", value, mode).Error; err != nil {
		t.Fatal(err)
	}
	return db
}
