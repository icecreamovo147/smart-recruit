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
	if owner.Type == model.OwnerTenant {
		return p.tenantEnabled(ctx, owner.ID, capability)
	}
	return p.userEnabled(ctx, owner.ID, capability)
}

func (p *EntitlementPolicy) tenantEnabled(ctx context.Context, tenantID uint64, capability string) (bool, error) {
	jsonPath := fmt.Sprintf(`$."%s"`, strings.ReplaceAll(capability, `"`, ``))
	var paidEnabled *bool
	if err := p.db.WithContext(ctx).Raw(`
		SELECT CAST(JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?)) AS UNSIGNED)
		FROM billing_subscriptions subscription
		JOIN billing_price_versions price ON price.id = subscription.price_version_id
		WHERE subscription.owner_type = 'tenant' AND subscription.owner_id = ? AND subscription.status = 'active'
		  AND subscription.current_period_start <= UTC_TIMESTAMP(3) AND subscription.current_period_end > UTC_TIMESTAMP(3)
		ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, jsonPath, tenantID).Scan(&paidEnabled).Error; err != nil {
		return false, err
	}
	if paidEnabled != nil {
		return *paidEnabled, nil
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
	var enabled *bool
	err := p.db.WithContext(ctx).Raw(`
		SELECT CAST(JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?)) AS UNSIGNED)
		FROM billing_subscriptions subscription
		JOIN billing_price_versions price ON price.id = subscription.price_version_id
		WHERE subscription.owner_type = 'user' AND subscription.owner_id = ? AND subscription.status = 'active'
		  AND subscription.current_period_start <= UTC_TIMESTAMP(3) AND subscription.current_period_end > UTC_TIMESTAMP(3)
		ORDER BY subscription.current_period_end DESC, subscription.id DESC LIMIT 1`, jsonPath, userID).Scan(&enabled).Error
	if err != nil {
		return false, err
	}
	if enabled != nil {
		return *enabled, nil
	}
	// Candidate Free is evergreen and lazily provisioned; it is intentionally limited
	// to capabilities explicitly present in its published entitlement snapshot.
	err = p.db.WithContext(ctx).Raw(`
		SELECT CAST(JSON_UNQUOTE(JSON_EXTRACT(price.entitlement_snapshot, ?)) AS UNSIGNED)
		FROM billing_products product
		JOIN billing_price_versions price ON price.product_id = product.id
		WHERE product.product_key = 'candidate_free' AND product.status = 'active' AND price.status = 'published'
		  AND price.effective_at <= UTC_TIMESTAMP(3)
		ORDER BY price.version DESC LIMIT 1`, jsonPath).Scan(&enabled).Error
	if err != nil {
		return false, err
	}
	return enabled != nil && *enabled, nil
}

var _ interface {
	Enabled(context.Context, model.Owner, string) (bool, error)
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
