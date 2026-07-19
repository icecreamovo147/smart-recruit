package quota

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

var ErrLimitExceeded = errors.New("tenant quota limit exceeded")

type Result struct {
	Allowed         bool
	MetricKey       string
	Usage           int64
	Limit           int64
	EnforcementMode string
}

type Checker struct {
	db  *gorm.DB
	now func() time.Time
}

func NewChecker(db *gorm.DB) *Checker { return &Checker{db: db, now: time.Now} }

func (c *Checker) Check(ctx context.Context, tenantID int64, metricKey string, delta int64) (Result, error) {
	result := Result{Allowed: true, MetricKey: metricKey}
	if c == nil || c.db == nil || tenantID <= 0 || delta < 0 {
		return result, nil
	}
	var entitlement struct {
		ValueJSON       string
		EnforcementMode string
	}
	err := c.db.WithContext(ctx).Raw(`
		SELECT COALESCE(overrides.value_json, entitlement.value_json) AS value_json,
		       entitlement.enforcement_mode
		FROM tenant_subscriptions subscription
		JOIN platform_plan_versions version ON version.id = subscription.plan_version_id
		JOIN platform_plan_entitlements entitlement ON entitlement.plan_version_id = version.id AND entitlement.entitlement_key = ?
		LEFT JOIN tenant_entitlement_overrides overrides ON overrides.tenant_id = subscription.tenant_id
		  AND overrides.entitlement_key = entitlement.entitlement_key
		  AND (overrides.expires_at IS NULL OR overrides.expires_at > ?)
		WHERE subscription.tenant_id = ? AND subscription.status = 'active'
		  AND subscription.starts_at <= ? AND (subscription.ends_at IS NULL OR subscription.ends_at > ?)
		ORDER BY subscription.starts_at DESC, subscription.id DESC LIMIT 1`, metricKey, c.now(), tenantID, c.now(), c.now()).Scan(&entitlement).Error
	if err != nil {
		return result, err
	}
	if entitlement.ValueJSON == "" {
		return result, nil
	}
	limit, err := integerJSON(entitlement.ValueJSON)
	if err != nil || limit <= 0 {
		return result, fmt.Errorf("invalid quota %s value: %w", metricKey, err)
	}
	usage, err := c.currentUsage(ctx, tenantID, metricKey)
	if err != nil {
		return result, err
	}
	result.Usage, result.Limit, result.EnforcementMode = usage, limit, entitlement.EnforcementMode
	result.Allowed = entitlement.EnforcementMode != "hard" || usage+delta <= limit
	if !result.Allowed {
		return result, ErrLimitExceeded
	}
	return result, nil
}

func (c *Checker) currentUsage(ctx context.Context, tenantID int64, metricKey string) (int64, error) {
	var count int64
	query := c.db.WithContext(ctx)
	countRows := func(db *gorm.DB) (int64, error) {
		err := db.Count(&count).Error
		return count, err
	}
	switch metricKey {
	case "members.max":
		return countRows(query.Table("tenant_memberships").Where("tenant_id = ? AND status = 'active'", tenantID))
	case "jobs.published.max":
		return countRows(query.Table("jobs").Where("tenant_id = ? AND status = 1", tenantID))
	case "applications.monthly.max":
		now := c.now()
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return countRows(query.Table("applications").Where("tenant_id = ? AND applied_at >= ?", tenantID, monthStart))
	case "resumes.storage.max":
		return countRows(query.Table("applications").Where("tenant_id = ?", tenantID).Distinct("resume_id"))
	default:
		return 0, fmt.Errorf("unsupported quota metric %q", metricKey)
	}
}

func integerJSON(value string) (int64, error) {
	var number json.Number
	if err := json.Unmarshal([]byte(value), &number); err == nil {
		return strconv.ParseInt(number.String(), 10, 64)
	}
	return strconv.ParseInt(value, 10, 64)
}
