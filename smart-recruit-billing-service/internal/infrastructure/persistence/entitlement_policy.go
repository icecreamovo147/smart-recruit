package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"smart-recruit-billing-service/internal/domain/model"
)

type EntitlementPolicy struct{ db *gorm.DB }

func NewEntitlementPolicy(db *gorm.DB) (*EntitlementPolicy, error) {
	if db == nil {
		return nil, errors.New("entitlement database is required")
	}
	return &EntitlementPolicy{db: db}, nil
}

func (p *EntitlementPolicy) Enabled(ctx context.Context, owner model.Owner, capability string) (bool, error) {
	if err := owner.Validate(); err != nil {
		return false, err
	}
	capability = strings.TrimSpace(capability)
	if capability == "" {
		return false, errors.New("AI capability is required")
	}
	capability = enabledEntitlementKey(capability)
	if owner.Type == model.OwnerTenant {
		return p.tenantEnabled(ctx, owner.ID, capability)
	}
	return p.userEnabled(ctx, owner.ID, capability)
}

// ReleaseVersionID resolves the immutable platform AI capability release from
// the same paid snapshot / platform-plan chain used for the enablement gate.
// A zero value means the entitlement exists without a release binding and is
// treated as unavailable by runtime callers once enforcement is enabled.
func (p *EntitlementPolicy) ReleaseVersionID(ctx context.Context, owner model.Owner, capability string) (uint64, error) {
	if err := owner.Validate(); err != nil {
		return 0, err
	}
	capability = strings.TrimSpace(capability)
	if capability == "" {
		return 0, errors.New("AI capability is required")
	}
	key := releaseVersionEntitlementKey(capability)
	if owner.Type == model.OwnerTenant {
		return p.tenantReleaseVersionID(ctx, owner.ID, key)
	}
	return p.userReleaseVersionID(ctx, owner.ID, key)
}

// ReservationCreditLimit resolves the commercial single-run ceiling from the
// same immutable paid snapshot / platform plan chain used for capability
// access. Missing legacy values return zero so the application service can use
// its compatibility default during rollout.
func (p *EntitlementPolicy) ReservationCreditLimit(ctx context.Context, owner model.Owner) (uint64, error) {
	if err := owner.Validate(); err != nil {
		return 0, err
	}
	const key = "ai.single_run.max_credits"
	if owner.Type == model.OwnerTenant {
		return p.tenantInteger(ctx, owner.ID, key)
	}
	return p.userInteger(ctx, owner.ID, key)
}

func (p *EntitlementPolicy) tenantInteger(ctx context.Context, tenantID uint64, key string) (uint64, error) {
	jsonPath := fmt.Sprintf(`$."%s"`, strings.ReplaceAll(key, `"`, ``))
	var paid *uint64
	if err := p.db.WithContext(ctx).Raw(`
		SELECT CAST(JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?)) AS UNSIGNED)
		FROM billing_subscriptions subscription
		JOIN billing_price_versions price ON price.id = subscription.price_version_id
		WHERE subscription.owner_type = 'tenant' AND subscription.owner_id = ? AND subscription.status = 'active'
		  AND subscription.current_period_start <= UTC_TIMESTAMP(3) AND subscription.current_period_end > UTC_TIMESTAMP(3)
		ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, jsonPath, tenantID).Scan(&paid).Error; err != nil {
		return 0, err
	}
	if paid != nil {
		return *paid, nil
	}
	var plan *uint64
	err := p.db.WithContext(ctx).Raw(`
		SELECT CAST(JSON_UNQUOTE(COALESCE(override_entitlement.value_json, plan_entitlement.value_json)) AS UNSIGNED)
		FROM tenant_subscriptions subscription
		JOIN platform_plan_entitlements plan_entitlement
		  ON plan_entitlement.plan_version_id = subscription.plan_version_id
		 AND plan_entitlement.entitlement_key = ?
		LEFT JOIN tenant_entitlement_overrides override_entitlement
		  ON override_entitlement.tenant_id = subscription.tenant_id
		 AND override_entitlement.entitlement_key = plan_entitlement.entitlement_key
		 AND (override_entitlement.expires_at IS NULL OR override_entitlement.expires_at > UTC_TIMESTAMP(3))
		WHERE subscription.tenant_id = ? AND subscription.status = 'active'
		  AND subscription.starts_at <= UTC_TIMESTAMP(3)
		  AND (subscription.ends_at IS NULL OR subscription.ends_at > UTC_TIMESTAMP(3))
		ORDER BY subscription.starts_at DESC, subscription.id DESC LIMIT 1`, key, tenantID).Scan(&plan).Error
	if err != nil || plan == nil {
		return 0, err
	}
	return *plan, nil
}

func (p *EntitlementPolicy) userInteger(ctx context.Context, userID uint64, key string) (uint64, error) {
	jsonPath := fmt.Sprintf(`$."%s"`, strings.ReplaceAll(key, `"`, ``))
	var value *uint64
	if err := p.db.WithContext(ctx).Raw(`
		SELECT CAST(JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?)) AS UNSIGNED)
		FROM billing_subscriptions subscription
		JOIN billing_price_versions price ON price.id = subscription.price_version_id
		WHERE subscription.owner_type = 'user' AND subscription.owner_id = ? AND subscription.status = 'active'
		  AND subscription.current_period_start <= UTC_TIMESTAMP(3) AND subscription.current_period_end > UTC_TIMESTAMP(3)
		ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, jsonPath, userID).Scan(&value).Error; err != nil {
		return 0, err
	}
	if value != nil {
		return *value, nil
	}
	err := p.db.WithContext(ctx).Raw(`
		SELECT CAST(JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?)) AS UNSIGNED)
		FROM billing_products product
		JOIN billing_price_versions price ON price.product_id = product.id
		WHERE product.product_key = 'candidate_free' AND product.status = 'active' AND price.status = 'published'
		  AND price.effective_at <= UTC_TIMESTAMP(3)
		ORDER BY price.version DESC LIMIT 1`, jsonPath).Scan(&value).Error
	if err != nil || value == nil {
		return 0, err
	}
	return *value, nil
}

func enabledEntitlementKey(capability string) string {
	capability = strings.TrimSpace(capability)
	if strings.HasSuffix(capability, ".enabled") {
		return capability
	}
	return capability + ".enabled"
}

func releaseVersionEntitlementKey(capability string) string {
	capability = strings.TrimSuffix(strings.TrimSpace(capability), ".enabled")
	return capability + ".release_version_id"
}

func (p *EntitlementPolicy) tenantEnabled(ctx context.Context, tenantID uint64, capability string) (bool, error) {
	jsonPath := fmt.Sprintf(`$."%s"`, strings.ReplaceAll(capability, `"`, ``))
	var paidEnabled *string
	if err := p.db.WithContext(ctx).Raw(`
		SELECT JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?))
		FROM billing_subscriptions subscription
		JOIN billing_price_versions price ON price.id = subscription.price_version_id
		WHERE subscription.owner_type = 'tenant' AND subscription.owner_id = ? AND subscription.status = 'active'
		  AND subscription.current_period_start <= UTC_TIMESTAMP(3) AND subscription.current_period_end > UTC_TIMESTAMP(3)
		ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, jsonPath, tenantID).Scan(&paidEnabled).Error; err != nil {
		return false, err
	}
	if paidEnabled != nil {
		return entitlementBoolean(*paidEnabled), nil
	}
	var value string
	err := p.db.WithContext(ctx).Raw(`
		SELECT CAST(COALESCE(override_entitlement.value_json, plan_entitlement.value_json) AS CHAR)
		FROM tenant_subscriptions subscription
		JOIN platform_plan_entitlements plan_entitlement
		  ON plan_entitlement.plan_version_id = subscription.plan_version_id
		 AND plan_entitlement.entitlement_key = ?
		LEFT JOIN tenant_entitlement_overrides override_entitlement
		  ON override_entitlement.tenant_id = subscription.tenant_id
		 AND override_entitlement.entitlement_key = plan_entitlement.entitlement_key
		 AND (override_entitlement.expires_at IS NULL OR override_entitlement.expires_at > UTC_TIMESTAMP(3))
		WHERE subscription.tenant_id = ? AND subscription.status = 'active'
		  AND subscription.starts_at <= UTC_TIMESTAMP(3)
		  AND (subscription.ends_at IS NULL OR subscription.ends_at > UTC_TIMESTAMP(3))
		ORDER BY subscription.starts_at DESC, subscription.id DESC LIMIT 1`, capability, tenantID).Scan(&value).Error
	if err != nil {
		return false, err
	}
	return value == "true" || value == "1", nil
}

func (p *EntitlementPolicy) userEnabled(ctx context.Context, userID uint64, capability string) (bool, error) {
	jsonPath := fmt.Sprintf(`$."%s"`, strings.ReplaceAll(capability, `"`, ``))
	var enabled *string
	err := p.db.WithContext(ctx).Raw(`
		SELECT JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?))
		FROM billing_subscriptions subscription
		JOIN billing_price_versions price ON price.id = subscription.price_version_id
		WHERE subscription.owner_type = 'user' AND subscription.owner_id = ? AND subscription.status = 'active'
		  AND subscription.current_period_start <= UTC_TIMESTAMP(3) AND subscription.current_period_end > UTC_TIMESTAMP(3)
		ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, jsonPath, userID).Scan(&enabled).Error
	if err != nil {
		return false, err
	}
	if enabled != nil {
		return entitlementBoolean(*enabled), nil
	}
	// Candidate Free is evergreen and lazily provisioned; it is intentionally limited
	// to capabilities explicitly present in its published entitlement snapshot.
	err = p.db.WithContext(ctx).Raw(`
		SELECT JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?))
		FROM billing_products product
		JOIN billing_price_versions price ON price.product_id = product.id
		WHERE product.product_key = 'candidate_free' AND product.status = 'active' AND price.status = 'published'
		  AND price.effective_at <= UTC_TIMESTAMP(3)
		ORDER BY price.version DESC LIMIT 1`, jsonPath).Scan(&enabled).Error
	if err != nil {
		return false, err
	}
	return enabled != nil && entitlementBoolean(*enabled), nil
}

func entitlementBoolean(value string) bool {
	value = strings.TrimSpace(value)
	return value == "1" || strings.EqualFold(value, "true")
}

func (p *EntitlementPolicy) tenantReleaseVersionID(ctx context.Context, tenantID uint64, entitlementKey string) (uint64, error) {
	jsonPath := fmt.Sprintf(`$."%s"`, strings.ReplaceAll(entitlementKey, `"`, ``))
	var paidVersion *uint64
	if err := p.db.WithContext(ctx).Raw(`
		SELECT CAST(JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?)) AS UNSIGNED)
		FROM billing_subscriptions subscription
		JOIN billing_price_versions price ON price.id = subscription.price_version_id
		WHERE subscription.owner_type = 'tenant' AND subscription.owner_id = ? AND subscription.status = 'active'
		  AND subscription.current_period_start <= UTC_TIMESTAMP(3) AND subscription.current_period_end > UTC_TIMESTAMP(3)
		ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, jsonPath, tenantID).Scan(&paidVersion).Error; err != nil {
		return 0, err
	}
	if paidVersion != nil {
		return *paidVersion, nil
	}
	var planVersion *uint64
	err := p.db.WithContext(ctx).Raw(`
		SELECT CAST(JSON_UNQUOTE(COALESCE(override_entitlement.value_json, plan_entitlement.value_json)) AS UNSIGNED)
		FROM tenant_subscriptions subscription
		JOIN platform_plan_entitlements plan_entitlement
		  ON plan_entitlement.plan_version_id = subscription.plan_version_id
		 AND plan_entitlement.entitlement_key = ?
		LEFT JOIN tenant_entitlement_overrides override_entitlement
		  ON override_entitlement.tenant_id = subscription.tenant_id
		 AND override_entitlement.entitlement_key = plan_entitlement.entitlement_key
		 AND (override_entitlement.expires_at IS NULL OR override_entitlement.expires_at > UTC_TIMESTAMP(3))
		WHERE subscription.tenant_id = ? AND subscription.status = 'active'
		  AND subscription.starts_at <= UTC_TIMESTAMP(3)
		  AND (subscription.ends_at IS NULL OR subscription.ends_at > UTC_TIMESTAMP(3))
		ORDER BY subscription.starts_at DESC, subscription.id DESC LIMIT 1`, entitlementKey, tenantID).Scan(&planVersion).Error
	if err != nil || planVersion == nil {
		return 0, err
	}
	return *planVersion, nil
}

func (p *EntitlementPolicy) userReleaseVersionID(ctx context.Context, userID uint64, entitlementKey string) (uint64, error) {
	jsonPath := fmt.Sprintf(`$."%s"`, strings.ReplaceAll(entitlementKey, `"`, ``))
	var version *uint64
	if err := p.db.WithContext(ctx).Raw(`
		SELECT CAST(JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?)) AS UNSIGNED)
		FROM billing_subscriptions subscription
		JOIN billing_price_versions price ON price.id = subscription.price_version_id
		WHERE subscription.owner_type = 'user' AND subscription.owner_id = ? AND subscription.status = 'active'
		  AND subscription.current_period_start <= UTC_TIMESTAMP(3) AND subscription.current_period_end > UTC_TIMESTAMP(3)
		ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, jsonPath, userID).Scan(&version).Error; err != nil {
		return 0, err
	}
	if version != nil {
		return *version, nil
	}
	err := p.db.WithContext(ctx).Raw(`
		SELECT CAST(JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?)) AS UNSIGNED)
		FROM billing_products product
		JOIN billing_price_versions price ON price.product_id = product.id
		WHERE product.product_key = 'candidate_free' AND product.status = 'active' AND price.status = 'published'
		  AND price.effective_at <= UTC_TIMESTAMP(3)
		ORDER BY price.version DESC LIMIT 1`, jsonPath).Scan(&version).Error
	if err != nil || version == nil {
		return 0, err
	}
	return *version, nil
}

var _ interface {
	Enabled(context.Context, model.Owner, string) (bool, error)
	ReleaseVersionID(context.Context, model.Owner, string) (uint64, error)
} = (*EntitlementPolicy)(nil)

func shanghaiMonth(now time.Time) (time.Time, time.Time) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	local := now.In(location)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
	return start.UTC(), start.AddDate(0, 1, 0).UTC()
}
