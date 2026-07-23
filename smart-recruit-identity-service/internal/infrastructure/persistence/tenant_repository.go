package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	commonsquota "smart-recruit-commons/quota"
	"smart-recruit-identity-service/internal/domain/model"
	platformmetadata "smart-recruit-platform-go/metadata"
)

type TenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) CheckTenantQuota(ctx context.Context, tenantID int64, metricKey string, delta int64) error {
	_, err := commonsquota.NewChecker(r.db).Check(ctx, tenantID, metricKey, delta)
	return err
}

func (r *TenantRepository) GetByID(ctx context.Context, tenantID int64) (*model.Tenant, error) {
	if tenantID <= 0 {
		return nil, nil
	}
	var row tenantRow
	err := r.db.WithContext(ctx).Where("id = ?", tenantID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *TenantRepository) GetDefault(ctx context.Context) (*model.Tenant, error) {
	var row tenantRow
	err := r.db.WithContext(ctx).Where("is_default = 1").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *TenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	row := tenantRow{TenantKey: tenant.TenantKey, Slug: tenant.Slug, Name: tenant.Name, Status: tenant.Status, Timezone: tenant.Timezone, Locale: tenant.Locale}
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return createPlatformAudit(tx, ctx, "tenant.create", row.ID, nil, tenantAuditSnapshot(row))
	}); err != nil {
		return err
	}
	tenant.ID = row.ID
	return nil
}

func (r *TenantRepository) List(ctx context.Context, offset, limit int, keyword, status string) ([]model.Tenant, int64, error) {
	query := r.db.WithContext(ctx).Model(&tenantRow{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR slug LIKE ? OR tenant_key LIKE ?", like, like, like)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []tenantRow
	if err := query.Select("tenants.*, (SELECT COUNT(*) FROM tenant_memberships membership WHERE membership.tenant_id = tenants.id) AS membership_count").Order("is_default DESC, created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]model.Tenant, len(rows))
	for i := range rows {
		result[i] = *rows[i].toDomain()
	}
	return result, total, nil
}

func (r *TenantRepository) GetPlatformDashboard(ctx context.Context) (model.PlatformDashboard, error) {
	var dashboard model.PlatformDashboard
	if err := r.db.WithContext(ctx).Model(&tenantRow{}).Count(&dashboard.TotalTenants).Error; err != nil {
		return dashboard, err
	}
	statusCounts := []struct {
		Status string
		Count  int64
	}{}
	if err := r.db.WithContext(ctx).Model(&tenantRow{}).Select("status, COUNT(*) AS count").Group("status").Scan(&statusCounts).Error; err != nil {
		return dashboard, err
	}
	for _, item := range statusCounts {
		switch item.Status {
		case "active":
			dashboard.ActiveTenants = item.Count
		case "suspended":
			dashboard.SuspendedTenants = item.Count
		case "disabled":
			dashboard.DisabledTenants = item.Count
		}
	}
	if err := r.db.WithContext(ctx).Model(&tenantRow{}).Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).Count(&dashboard.NewTenants30D).Error; err != nil {
		return dashboard, err
	}
	if err := r.db.WithContext(ctx).Table("tenant_memberships").Count(&dashboard.TotalMemberships).Error; err != nil {
		return dashboard, err
	}
	if err := r.db.WithContext(ctx).Table("tenant_memberships").Where("status = ?", "active").Count(&dashboard.ActiveMemberships).Error; err != nil {
		return dashboard, err
	}
	if err := r.db.WithContext(ctx).Table("tenants tenant").
		Where(`NOT EXISTS (
			SELECT 1 FROM tenant_memberships membership
			JOIN tenant_membership_roles assignment ON assignment.membership_id = membership.id AND assignment.revoked_at IS NULL
			JOIN roles role ON role.id = assignment.role_id AND role.role_key = ?
			WHERE membership.tenant_id = tenant.id AND membership.status = 'active'
		)`, model.RoleRecruitingAdmin).
		Count(&dashboard.TenantsWithoutAdmin).Error; err != nil {
		return dashboard, err
	}
	return dashboard, nil
}

func (r *TenantRepository) QueryPlatformAuditLogs(ctx context.Context, filter model.PlatformAuditFilter, offset, limit int) ([]model.PlatformAuditLog, int64, error) {
	query := r.db.WithContext(ctx).Table("platform_audit_logs audit")
	if filter.TenantID > 0 {
		query = query.Where("audit.target_tenant_id = ?", filter.TenantID)
	}
	if filter.ActorUserID > 0 {
		query = query.Where("audit.actor_user_id = ?", filter.ActorUserID)
	}
	if filter.Action != "" {
		query = query.Where("audit.action = ?", filter.Action)
	}
	if filter.RequestID != "" {
		query = query.Where("audit.request_id = ?", filter.RequestID)
	}
	if filter.StartTime != nil {
		query = query.Where("audit.created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("audit.created_at < ?", *filter.EndTime)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []platformAuditListRow
	if err := query.Select(`audit.id, audit.actor_user_id, COALESCE(actor.username, '') AS actor_username,
		audit.action, audit.resource_type, audit.resource_id, COALESCE(audit.target_tenant_id, 0) AS target_tenant_id,
		COALESCE(tenant.name, '') AS target_tenant_name, audit.before_json, audit.after_json,
		audit.request_id, audit.client_ip, audit.created_at`).
		Joins("LEFT JOIN users actor ON actor.id = audit.actor_user_id").
		Joins("LEFT JOIN tenants tenant ON tenant.id = audit.target_tenant_id").
		Order("audit.created_at DESC, audit.id DESC").Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]model.PlatformAuditLog, len(rows))
	for i := range rows {
		result[i] = rows[i].toDomain()
	}
	return result, total, nil
}

func (r *TenantRepository) ListPlatformPlans(ctx context.Context, status string) ([]model.PlatformPlan, error) {
	query := r.db.WithContext(ctx).Table("platform_plans")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var rows []platformPlanRow
	if err := query.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	plans := make([]model.PlatformPlan, len(rows))
	for i, row := range rows {
		plans[i] = row.toDomain()
		versions, err := r.listPlanVersions(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		plans[i].Versions = versions
	}
	return plans, nil
}

func (r *TenantRepository) listPlanVersions(ctx context.Context, planID int64) ([]model.PlatformPlanVersion, error) {
	var rows []platformPlanVersionRow
	if err := r.db.WithContext(ctx).Where("plan_id = ?", planID).Order("version DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	versions := make([]model.PlatformPlanVersion, len(rows))
	for i, row := range rows {
		versions[i] = row.toDomain()
		entitlements, err := r.listEntitlements(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		versions[i].Entitlements = entitlements
	}
	return versions, nil
}

func (r *TenantRepository) listEntitlements(ctx context.Context, versionID int64) ([]model.PlatformEntitlement, error) {
	var rows []platformEntitlementRow
	if err := r.db.WithContext(ctx).Where("plan_version_id = ?", versionID).Order("entitlement_key ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]model.PlatformEntitlement, len(rows))
	for i, row := range rows {
		result[i] = model.PlatformEntitlement{Key: row.EntitlementKey, ValueType: row.ValueType, ValueJSON: row.ValueJSON, EnforcementMode: row.EnforcementMode, Source: "plan"}
	}
	return result, nil
}

func (r *TenantRepository) SavePlatformPlanVersion(ctx context.Context, planID, versionID int64, changeNote string, entitlements []model.PlatformEntitlement) (*model.PlatformPlanVersion, error) {
	var savedID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var plan platformPlanRow
		if err := tx.Where("id = ? AND status = 'active'", planID).First(&plan).Error; err != nil {
			return err
		}
		var version platformPlanVersionRow
		if versionID == 0 {
			var latest int32
			if err := tx.Model(&platformPlanVersionRow{}).Where("plan_id = ?", planID).Select("COALESCE(MAX(version), 0)").Scan(&latest).Error; err != nil {
				return err
			}
			version = platformPlanVersionRow{PlanID: planID, Version: latest + 1, Status: "draft", ChangeNote: changeNote, CreatedBy: platformmetadata.GetAuthUserID(ctx)}
			if err := tx.Create(&version).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND plan_id = ? AND status = 'draft'", versionID, planID).First(&version).Error; err != nil {
				return err
			}
			if err := tx.Model(&version).Update("change_note", changeNote).Error; err != nil {
				return err
			}
			if err := tx.Where("plan_version_id = ?", version.ID).Delete(&platformEntitlementRow{}).Error; err != nil {
				return err
			}
		}
		for _, entitlement := range entitlements {
			if strings.HasSuffix(entitlement.Key, ".release_version_id") {
				releaseID, err := strconv.ParseInt(entitlement.ValueJSON, 10, 64)
				if err != nil || releaseID <= 0 {
					return errors.New("AI capability release entitlement is invalid")
				}
				capabilityKey := strings.TrimSuffix(entitlement.Key, ".release_version_id")
				var count int64
				if err := tx.Table("platform_ai_capability_versions version").
					Joins("JOIN platform_ai_capabilities capability ON capability.id = version.capability_id").
					Where("version.id = ? AND version.status = 'published' AND capability.status = 'active' AND capability.audience = 'tenant_hr' AND capability.capability_key = ?", releaseID, capabilityKey).
					Count(&count).Error; err != nil {
					return err
				}
				if count != 1 {
					return errors.New("AI capability release entitlement must reference a published tenant release")
				}
			}
			row := platformEntitlementRow{PlanVersionID: version.ID, EntitlementKey: entitlement.Key, ValueType: entitlement.ValueType, ValueJSON: entitlement.ValueJSON, EnforcementMode: entitlement.EnforcementMode}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		savedID = version.ID
		return createPlatformResourceAudit(tx, ctx, "plan.version.save", "plan_version", version.ID, 0, nil, map[string]any{"plan_id": planID, "version": version.Version, "status": "draft", "change_note": changeNote})
	})
	if err != nil {
		return nil, err
	}
	return r.getPlanVersion(ctx, savedID)
}

func (r *TenantRepository) PublishPlatformPlanVersion(ctx context.Context, planID, versionID int64, effectiveAt time.Time, reason string) (*model.PlatformPlanVersion, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var version platformPlanVersionRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND plan_id = ? AND status = 'draft'", versionID, planID).First(&version).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&platformEntitlementRow{}).Where("plan_version_id = ?", versionID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return errors.New("plan version has no entitlements")
		}
		if err := tx.Model(&version).Updates(map[string]any{"status": "published", "effective_at": effectiveAt, "published_by": platformmetadata.GetAuthUserID(ctx)}).Error; err != nil {
			return err
		}
		return createPlatformResourceAudit(tx, ctx, "plan.version.publish", "plan_version", version.ID, 0, map[string]any{"status": "draft"}, map[string]any{"status": "published", "effective_at": effectiveAt, "reason": reason})
	})
	if err != nil {
		return nil, err
	}
	return r.getPlanVersion(ctx, versionID)
}

func (r *TenantRepository) getPlanVersion(ctx context.Context, versionID int64) (*model.PlatformPlanVersion, error) {
	var row platformPlanVersionRow
	if err := r.db.WithContext(ctx).Where("id = ?", versionID).First(&row).Error; err != nil {
		return nil, err
	}
	result := row.toDomain()
	entitlements, err := r.listEntitlements(ctx, versionID)
	if err != nil {
		return nil, err
	}
	result.Entitlements = entitlements
	return &result, nil
}

func (r *TenantRepository) GetTenantSubscription(ctx context.Context, tenantID int64) (*model.TenantSubscription, error) {
	now := time.Now()
	var row tenantSubscriptionListRow
	err := r.db.WithContext(ctx).Table("tenant_subscriptions subscription").
		Select("subscription.*, plan.plan_key, plan.name AS plan_name, version.version AS plan_version").
		Joins("JOIN platform_plan_versions version ON version.id = subscription.plan_version_id").
		Joins("JOIN platform_plans plan ON plan.id = version.plan_id").
		Where("subscription.tenant_id = ? AND subscription.status IN ('active', 'scheduled') AND subscription.starts_at <= ? AND (subscription.ends_at IS NULL OR subscription.ends_at > ?)", tenantID, now, now).
		Order("subscription.starts_at DESC, subscription.id DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := row.toDomain()
	result.EffectiveEntitlements, err = r.effectiveEntitlements(ctx, tenantID, row.PlanVersionID)
	return &result, err
}

func (r *TenantRepository) UpdateTenantSubscription(ctx context.Context, tenantID, planVersionID int64, startsAt time.Time, endsAt *time.Time, reason string) (*model.TenantSubscription, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var version platformPlanVersionRow
		if err := tx.Where("id = ? AND status = 'published' AND effective_at <= ?", planVersionID, startsAt).First(&version).Error; err != nil {
			return err
		}
		if err := tx.Model(&tenantSubscriptionRow{}).Where("tenant_id = ? AND status IN ('active', 'scheduled')", tenantID).
			Updates(map[string]any{"status": "cancelled", "ends_at": time.Now()}).Error; err != nil {
			return err
		}
		row := tenantSubscriptionRow{TenantID: tenantID, PlanVersionID: planVersionID, Status: "active", StartsAt: startsAt, EndsAt: endsAt, Reason: reason, CreatedBy: platformmetadata.GetAuthUserID(ctx)}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return createPlatformResourceAudit(tx, ctx, "tenant.subscription.update", "tenant_subscription", row.ID, tenantID, nil, map[string]any{"plan_version_id": planVersionID, "starts_at": startsAt, "ends_at": endsAt, "reason": reason})
	})
	if err != nil {
		return nil, err
	}
	return r.GetTenantSubscription(ctx, tenantID)
}

func (r *TenantRepository) UpdateTenantEntitlementOverride(ctx context.Context, tenantID int64, entitlement model.PlatformEntitlement, expiresAt *time.Time, reason string) (*model.TenantSubscription, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before tenantEntitlementOverrideRow
		beforeErr := tx.Where("tenant_id = ? AND entitlement_key = ?", tenantID, entitlement.Key).First(&before).Error
		if beforeErr != nil && !errors.Is(beforeErr, gorm.ErrRecordNotFound) {
			return beforeErr
		}
		row := tenantEntitlementOverrideRow{TenantID: tenantID, EntitlementKey: entitlement.Key, ValueType: entitlement.ValueType, ValueJSON: entitlement.ValueJSON, Reason: reason, ExpiresAt: expiresAt, CreatedBy: platformmetadata.GetAuthUserID(ctx)}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "entitlement_key"}},
			DoUpdates: clause.Assignments(map[string]any{"value_type": entitlement.ValueType, "value_json": entitlement.ValueJSON, "reason": reason, "expires_at": expiresAt, "updated_at": time.Now()}),
		}).Create(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 {
			if err := tx.Select("id").Where("tenant_id = ? AND entitlement_key = ?", tenantID, entitlement.Key).First(&row).Error; err != nil {
				return err
			}
		}
		var beforeSnapshot map[string]any
		if beforeErr == nil {
			beforeSnapshot = map[string]any{"value_json": before.ValueJSON, "expires_at": before.ExpiresAt, "reason": before.Reason}
		}
		return createPlatformResourceAudit(tx, ctx, "tenant.entitlement.override", "tenant_entitlement_override", row.ID, tenantID, beforeSnapshot, map[string]any{"entitlement_key": entitlement.Key, "value_json": entitlement.ValueJSON, "expires_at": expiresAt, "reason": reason})
	})
	if err != nil {
		return nil, err
	}
	return r.GetTenantSubscription(ctx, tenantID)
}

func (r *TenantRepository) effectiveEntitlements(ctx context.Context, tenantID, versionID int64) ([]model.PlatformEntitlement, error) {
	result, err := r.listEntitlements(ctx, versionID)
	if err != nil {
		return nil, err
	}
	var overrides []tenantEntitlementOverrideRow
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND (expires_at IS NULL OR expires_at > ?)", tenantID, time.Now()).Find(&overrides).Error; err != nil {
		return nil, err
	}
	byKey := make(map[string]int, len(result))
	for i := range result {
		byKey[result[i].Key] = i
	}
	for _, override := range overrides {
		item := model.PlatformEntitlement{Key: override.EntitlementKey, ValueType: override.ValueType, ValueJSON: override.ValueJSON, EnforcementMode: "hard", Source: "override"}
		if index, ok := byKey[item.Key]; ok {
			result[index] = item
		} else {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *TenantRepository) GetTenantUsage(ctx context.Context, tenantID int64) ([]model.TenantUsageMetric, error) {
	subscription, err := r.GetTenantSubscription(ctx, tenantID)
	if err != nil || subscription == nil {
		return nil, err
	}
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	usage := map[string]int64{}
	queries := []struct {
		key   string
		table string
		where string
		args  []any
	}{
		{key: "members.max", table: "tenant_memberships", where: "tenant_id = ? AND status = 'active'", args: []any{tenantID}},
		{key: "jobs.published.max", table: "jobs", where: "tenant_id = ? AND status = 1", args: []any{tenantID}},
		{key: "applications.monthly.max", table: "applications", where: "tenant_id = ? AND applied_at >= ?", args: []any{tenantID, monthStart}},
	}
	for _, query := range queries {
		var count int64
		if err := r.db.WithContext(ctx).Table(query.table).Where(query.where, query.args...).Count(&count).Error; err != nil {
			return nil, err
		}
		usage[query.key] = count
	}
	var resumeCount int64
	if err := r.db.WithContext(ctx).Table("applications").Where("tenant_id = ?", tenantID).Distinct("resume_id").Count(&resumeCount).Error; err != nil {
		return nil, err
	}
	usage["resumes.storage.max"] = resumeCount
	metrics := make([]model.TenantUsageMetric, 0, len(subscription.EffectiveEntitlements))
	for _, entitlement := range subscription.EffectiveEntitlements {
		quota, err := entitlementInteger(entitlement.ValueJSON)
		if err != nil || quota <= 0 {
			continue
		}
		value, supported := usage[entitlement.Key]
		if !supported {
			continue
		}
		percent := int32(value * 100 / quota)
		metrics = append(metrics, model.TenantUsageMetric{Key: entitlement.Key, UsageValue: value, QuotaValue: quota, UsagePercent: percent, EnforcementMode: entitlement.EnforcementMode, MeasuredAt: now})
		windowStart, windowEnd := usageWindow(entitlement.Key, now)
		if err := r.persistUsageSnapshot(ctx, tenantID, entitlement.Key, value, windowStart, windowEnd, now); err != nil {
			return nil, err
		}
		if err := r.refreshQuotaAlerts(ctx, tenantID, entitlement.Key, value, quota, percent); err != nil {
			return nil, err
		}
	}
	return metrics, nil
}

func (r *TenantRepository) RefreshQuotaUsage(ctx context.Context) error {
	now := time.Now()
	var tenantIDs []int64
	if err := r.db.WithContext(ctx).Table("tenant_subscriptions").
		Where("status IN ('active', 'scheduled') AND starts_at <= ? AND (ends_at IS NULL OR ends_at > ?)", now, now).
		Distinct("tenant_id").Pluck("tenant_id", &tenantIDs).Error; err != nil {
		return err
	}
	var refreshErrors []error
	for _, tenantID := range tenantIDs {
		if _, err := r.GetTenantUsage(ctx, tenantID); err != nil {
			refreshErrors = append(refreshErrors, fmt.Errorf("tenant %d: %w", tenantID, err))
		}
	}
	return errors.Join(refreshErrors...)
}

func usageWindow(metricKey string, measuredAt time.Time) (time.Time, time.Time) {
	if metricKey == "applications.monthly.max" {
		start := time.Date(measuredAt.Year(), measuredAt.Month(), 1, 0, 0, 0, 0, measuredAt.Location())
		return start, start.AddDate(0, 1, 0)
	}
	start := time.Date(measuredAt.Year(), measuredAt.Month(), measuredAt.Day(), 0, 0, 0, 0, measuredAt.Location())
	return start, start.AddDate(0, 0, 1)
}

func (r *TenantRepository) persistUsageSnapshot(ctx context.Context, tenantID int64, metricKey string, value int64, windowStart, windowEnd, measuredAt time.Time) error {
	row := tenantUsageSnapshotRow{TenantID: tenantID, MetricKey: metricKey, MetricValue: value, WindowStart: windowStart, WindowEnd: windowEnd, MeasuredAt: measuredAt}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "metric_key"}, {Name: "window_start"}, {Name: "window_end"}},
		DoUpdates: clause.Assignments(map[string]any{"metric_value": value, "measured_at": measuredAt}),
	}).Create(&row).Error
}

func entitlementInteger(value string) (int64, error) {
	var number json.Number
	if err := json.Unmarshal([]byte(value), &number); err == nil {
		return strconv.ParseInt(number.String(), 10, 64)
	}
	return strconv.ParseInt(value, 10, 64)
}

func (r *TenantRepository) refreshQuotaAlerts(ctx context.Context, tenantID int64, metricKey string, usage, quota int64, percent int32) error {
	for _, threshold := range []int32{80, 90, 100} {
		if percent < threshold {
			continue
		}
		var existing quotaAlertRow
		err := r.db.WithContext(ctx).Where("tenant_id = ? AND metric_key = ? AND threshold_percent = ? AND status IN ('open', 'acknowledged')", tenantID, metricKey, threshold).First(&existing).Error
		if err == nil {
			if err := r.db.WithContext(ctx).Model(&existing).Updates(map[string]any{"usage_value": usage, "quota_value": quota, "last_triggered_at": time.Now()}).Error; err != nil {
				return err
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row := quotaAlertRow{TenantID: tenantID, MetricKey: metricKey, ThresholdPercent: threshold, UsageValue: usage, QuotaValue: quota, Status: "open", FirstTriggeredAt: time.Now(), LastTriggeredAt: time.Now()}
		if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			return enqueueQuotaAlertEmails(tx, tenantID, row.ID, metricKey, threshold, usage, quota)
		}); err != nil {
			return err
		}
	}
	return nil
}

func enqueueQuotaAlertEmails(tx *gorm.DB, tenantID, alertID int64, metricKey string, threshold int32, usage, quota int64) error {
	var recipients []struct {
		ID       int64
		Username string
	}
	if err := tx.Table("users user").Select("DISTINCT user.id, user.username").
		Joins("JOIN platform_user_roles assignment ON assignment.user_id = user.id AND assignment.revoked_at IS NULL").
		Joins("JOIN roles role ON role.id = assignment.role_id").
		Where("user.status = 'active' AND user.email <> '' AND role.role_key IN ?", []string{model.RolePlatformAdmin, model.RolePlatformOperator}).Scan(&recipients).Error; err != nil {
		return err
	}
	for _, recipient := range recipients {
		eventID := uuid.NewString()
		idempotencyKey := fmt.Sprintf("quota-alert:%d:%d:email:%d", alertID, threshold, recipient.ID)
		payload, err := json.Marshal(map[string]any{
			"event_id": eventID, "event_type": "email.send", "idempotency_key": idempotencyKey,
			"receiver_id": recipient.ID, "receiver_account_type": "platform", "type": "platform_quota_alert",
			"title":   fmt.Sprintf("租户配额告警（%d%%）", threshold),
			"content": fmt.Sprintf("租户 %d 的 %s 用量已达 %d/%d，请及时处理。", tenantID, metricKey, usage, quota),
			"link":    "/quota-alerts", "biz_type": "quota_alert", "biz_id": alertID, "recipient_name": recipient.Username,
		})
		if err != nil {
			return err
		}
		row := platformOutboxRow{TenantID: &tenantID, EventID: eventID, SchemaVersion: "1.0", EventType: "email.send", AggregateType: "quota_alert", AggregateID: uint64(alertID), RoutingKey: "email.send", Producer: "identity-service", IdempotencyKey: idempotencyKey, Payload: string(payload), Metadata: "{}", Status: 0}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *TenantRepository) ListQuotaAlerts(ctx context.Context, filter model.QuotaAlertFilter, offset, limit int) ([]model.QuotaAlert, int64, error) {
	query := r.db.WithContext(ctx).Table("platform_quota_alerts alert")
	if filter.TenantID > 0 {
		query = query.Where("alert.tenant_id = ?", filter.TenantID)
	}
	if filter.Status != "" {
		query = query.Where("alert.status = ?", filter.Status)
	}
	if filter.MetricKey != "" {
		query = query.Where("alert.metric_key = ?", filter.MetricKey)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []quotaAlertListRow
	if err := query.Select("alert.*, tenant.name AS tenant_name").Joins("JOIN tenants tenant ON tenant.id = alert.tenant_id").Order("alert.last_triggered_at DESC, alert.id DESC").Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]model.QuotaAlert, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, total, nil
}

func (r *TenantRepository) UpdateQuotaAlert(ctx context.Context, alertID int64, status string, assigneeUserID int64, resolutionNote string) (*model.QuotaAlert, error) {
	var tenantID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before quotaAlertRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", alertID).First(&before).Error; err != nil {
			return err
		}
		tenantID = before.TenantID
		updates := map[string]any{"status": status, "assignee_user_id": nil, "resolution_note": resolutionNote}
		if assigneeUserID > 0 {
			updates["assignee_user_id"] = assigneeUserID
		}
		if status == "acknowledged" {
			updates["acknowledged_at"] = time.Now()
		}
		if status == "resolved" {
			updates["resolved_at"] = time.Now()
		}
		if err := tx.Model(&before).Updates(updates).Error; err != nil {
			return err
		}
		return createPlatformResourceAudit(tx, ctx, "quota_alert.status.update", "quota_alert", alertID, before.TenantID, map[string]any{"status": before.Status}, map[string]any{"status": status, "assignee_user_id": assigneeUserID, "resolution_note": resolutionNote})
	})
	if err != nil {
		return nil, err
	}
	items, _, err := r.ListQuotaAlerts(ctx, model.QuotaAlertFilter{TenantID: tenantID}, 0, 100)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == alertID {
			return &items[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *TenantRepository) UpdateStatus(ctx context.Context, tenantID int64, status, reason string) (*model.Tenant, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before tenantRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND is_default = 0", tenantID).First(&before).Error; err != nil {
			return err
		}
		if before.Status == status {
			return nil
		}
		if err := tx.Model(&tenantRow{}).Where("id = ? AND is_default = 0", tenantID).Update("status", status).Error; err != nil {
			return err
		}
		after := before
		after.Status = status
		afterSnapshot := tenantAuditSnapshot(after)
		afterSnapshot["reason"] = reason
		return createPlatformAudit(tx, ctx, "tenant.status.update", tenantID, tenantAuditSnapshot(before), afterSnapshot)
	})
	if err != nil {
		return nil, fmt.Errorf("update tenant status: %w", err)
	}
	return r.GetByID(ctx, tenantID)
}

func createPlatformAudit(tx *gorm.DB, ctx context.Context, action string, tenantID int64, before, after map[string]any) error {
	return createPlatformResourceAudit(tx, ctx, action, "tenant", tenantID, tenantID, before, after)
}

func createPlatformResourceAudit(tx *gorm.DB, ctx context.Context, action, resourceType string, resourceID, tenantID int64, before, after map[string]any) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	var targetTenantID *int64
	if tenantID > 0 {
		targetTenantID = &tenantID
	}
	row := platformAuditRow{
		ActorUserID:    platformmetadata.GetAuthUserID(ctx),
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		TargetTenantID: targetTenantID,
		BeforeJSON:     nullableAuditJSON(before, beforeJSON),
		AfterJSON:      nullableAuditJSON(after, afterJSON),
		RequestID:      platformmetadata.GetRequestID(ctx),
		ClientIP:       platformmetadata.GetClientIP(ctx),
	}
	return tx.Create(&row).Error
}

func nullableAuditJSON(value map[string]any, encoded []byte) *string {
	if value == nil {
		return nil
	}
	result := string(encoded)
	return &result
}

func tenantAuditSnapshot(row tenantRow) map[string]any {
	return map[string]any{"id": row.ID, "tenant_key": row.TenantKey, "slug": row.Slug, "name": row.Name, "status": row.Status, "timezone": row.Timezone, "locale": row.Locale, "is_default": row.IsDefault}
}

type platformAuditRow struct {
	ID             uint64    `gorm:"primaryKey"`
	ActorUserID    int64     `gorm:"column:actor_user_id"`
	Action         string    `gorm:"column:action"`
	ResourceType   string    `gorm:"column:resource_type"`
	ResourceID     int64     `gorm:"column:resource_id"`
	TargetTenantID *int64    `gorm:"column:target_tenant_id"`
	BeforeJSON     *string   `gorm:"column:before_json"`
	AfterJSON      *string   `gorm:"column:after_json"`
	RequestID      string    `gorm:"column:request_id"`
	ClientIP       string    `gorm:"column:client_ip"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (platformAuditRow) TableName() string { return "platform_audit_logs" }

type platformAuditListRow struct {
	ID               uint64
	ActorUserID      int64
	ActorUsername    string
	Action           string
	ResourceType     string
	ResourceID       int64
	TargetTenantID   int64
	TargetTenantName string
	BeforeJSON       *string
	AfterJSON        *string
	RequestID        string
	ClientIP         string
	CreatedAt        time.Time
}

type platformPlanRow struct {
	ID          int64
	PlanKey     string
	Name        string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (platformPlanRow) TableName() string { return "platform_plans" }
func (r platformPlanRow) toDomain() model.PlatformPlan {
	return model.PlatformPlan{ID: r.ID, PlanKey: r.PlanKey, Name: r.Name, Description: r.Description, Status: r.Status, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

type platformPlanVersionRow struct {
	ID          int64
	PlanID      int64
	Version     int32
	Status      string
	EffectiveAt *time.Time
	RetiredAt   *time.Time
	ChangeNote  string
	CreatedBy   int64
	PublishedBy int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (platformPlanVersionRow) TableName() string { return "platform_plan_versions" }
func (r platformPlanVersionRow) toDomain() model.PlatformPlanVersion {
	return model.PlatformPlanVersion{ID: r.ID, PlanID: r.PlanID, Version: r.Version, Status: r.Status, EffectiveAt: r.EffectiveAt, RetiredAt: r.RetiredAt, ChangeNote: r.ChangeNote, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

type platformEntitlementRow struct {
	ID              int64
	PlanVersionID   int64
	EntitlementKey  string
	ValueType       string
	ValueJSON       string `gorm:"type:json"`
	EnforcementMode string
}

func (platformEntitlementRow) TableName() string { return "platform_plan_entitlements" }

type tenantSubscriptionRow struct {
	ID            int64
	TenantID      int64
	PlanVersionID int64
	Status        string
	StartsAt      time.Time
	EndsAt        *time.Time
	Reason        string
	CreatedBy     int64
}

func (tenantSubscriptionRow) TableName() string { return "tenant_subscriptions" }

type tenantSubscriptionListRow struct {
	ID            int64
	TenantID      int64
	PlanVersionID int64
	Status        string
	StartsAt      time.Time
	EndsAt        *time.Time
	Reason        string
	CreatedBy     int64
	PlanKey       string
	PlanName      string
	PlanVersion   int32
}

func (r tenantSubscriptionListRow) toDomain() model.TenantSubscription {
	return model.TenantSubscription{ID: r.ID, TenantID: r.TenantID, PlanVersionID: r.PlanVersionID, PlanKey: r.PlanKey, PlanName: r.PlanName, PlanVersion: r.PlanVersion, Status: r.Status, StartsAt: r.StartsAt, EndsAt: r.EndsAt, Reason: r.Reason}
}

type tenantEntitlementOverrideRow struct {
	ID             int64
	TenantID       int64
	EntitlementKey string
	ValueType      string
	ValueJSON      string `gorm:"type:json"`
	Reason         string
	ExpiresAt      *time.Time
	CreatedBy      int64
}

func (tenantEntitlementOverrideRow) TableName() string { return "tenant_entitlement_overrides" }

type tenantUsageSnapshotRow struct {
	ID          int64
	TenantID    int64
	MetricKey   string
	MetricValue int64
	WindowStart time.Time
	WindowEnd   time.Time
	MeasuredAt  time.Time
}

func (tenantUsageSnapshotRow) TableName() string { return "tenant_usage_snapshots" }

type quotaAlertRow struct {
	ID               int64
	TenantID         int64
	MetricKey        string
	ThresholdPercent int32
	UsageValue       int64
	QuotaValue       int64
	Status           string
	AssigneeUserID   *int64
	AcknowledgedAt   *time.Time
	ResolvedAt       *time.Time
	ResolutionNote   string
	FirstTriggeredAt time.Time
	LastTriggeredAt  time.Time
}

func (quotaAlertRow) TableName() string { return "platform_quota_alerts" }

type quotaAlertListRow struct {
	ID               int64
	TenantID         int64
	TenantName       string
	MetricKey        string
	ThresholdPercent int32
	UsageValue       int64
	QuotaValue       int64
	Status           string
	AssigneeUserID   *int64
	AcknowledgedAt   *time.Time
	ResolvedAt       *time.Time
	ResolutionNote   string
	FirstTriggeredAt time.Time
	LastTriggeredAt  time.Time
}

type platformOutboxRow struct {
	ID             uint64 `gorm:"primaryKey"`
	TenantID       *int64
	EventID        string
	SchemaVersion  string
	EventType      string
	AggregateType  string
	AggregateID    uint64
	RoutingKey     string
	Producer       string
	IdempotencyKey string
	Payload        string `gorm:"type:json"`
	Metadata       string `gorm:"type:json"`
	Status         int32
}

func (platformOutboxRow) TableName() string { return "event_outbox" }

func (r quotaAlertListRow) toDomain() model.QuotaAlert {
	var assignee int64
	if r.AssigneeUserID != nil {
		assignee = *r.AssigneeUserID
	}
	return model.QuotaAlert{ID: r.ID, TenantID: r.TenantID, TenantName: r.TenantName, MetricKey: r.MetricKey, ThresholdPercent: r.ThresholdPercent, UsageValue: r.UsageValue, QuotaValue: r.QuotaValue, Status: r.Status, AssigneeUserID: assignee, AcknowledgedAt: r.AcknowledgedAt, ResolvedAt: r.ResolvedAt, ResolutionNote: r.ResolutionNote, FirstTriggeredAt: r.FirstTriggeredAt, LastTriggeredAt: r.LastTriggeredAt}
}

func (r platformAuditListRow) toDomain() model.PlatformAuditLog {
	result := model.PlatformAuditLog{
		ID: r.ID, ActorUserID: r.ActorUserID, ActorUsername: r.ActorUsername,
		Action: r.Action, ResourceType: r.ResourceType, ResourceID: r.ResourceID,
		TargetTenantID: r.TargetTenantID, TargetTenantName: r.TargetTenantName,
		RequestID: r.RequestID, ClientIP: r.ClientIP, CreatedAt: r.CreatedAt,
	}
	if r.BeforeJSON != nil {
		result.BeforeJSON = *r.BeforeJSON
	}
	if r.AfterJSON != nil {
		result.AfterJSON = *r.AfterJSON
	}
	return result
}

func (r *TenantRepository) ListTenantMemberships(ctx context.Context, tenantID int64, offset, limit int) ([]model.TenantMembership, int64, error) {
	query := r.db.WithContext(ctx).Table("tenant_memberships membership").Where("membership.tenant_id = ?", tenantID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []membershipTenantRow
	if err := query.Select("membership.id, membership.tenant_id, membership.user_id, membership.status, membership.joined_at, user.username, user.email, tenant.tenant_key, tenant.slug, tenant.name AS tenant_name, tenant.status AS tenant_status, tenant.timezone, tenant.locale, tenant.is_default").
		Joins("JOIN users user ON user.id = membership.user_id").Joins("JOIN tenants tenant ON tenant.id = membership.tenant_id").
		Order("membership.created_at DESC, membership.id DESC").Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]model.TenantMembership, len(rows))
	for i := range rows {
		result[i] = rows[i].toDomain()
		roles, err := r.membershipRoles(ctx, rows[i].ID)
		if err != nil {
			return nil, 0, err
		}
		result[i].Roles = roles
	}
	return result, total, nil
}

func (r *TenantRepository) GetTenantMembership(ctx context.Context, tenantID, membershipID int64) (*model.TenantMembership, error) {
	var row membershipTenantRow
	err := r.db.WithContext(ctx).Table("tenant_memberships membership").
		Select("membership.id, membership.tenant_id, membership.user_id, membership.status, membership.joined_at, user.username, user.email, tenant.tenant_key, tenant.slug, tenant.name AS tenant_name, tenant.status AS tenant_status, tenant.timezone, tenant.locale, tenant.is_default").
		Joins("JOIN users user ON user.id = membership.user_id").Joins("JOIN tenants tenant ON tenant.id = membership.tenant_id").
		Where("membership.tenant_id = ? AND membership.id = ?", tenantID, membershipID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	membership := row.toDomain()
	membership.Roles, err = r.membershipRoles(ctx, membershipID)
	if err != nil {
		return nil, err
	}
	return &membership, nil
}

func (r *TenantRepository) UpdateTenantMembershipStatus(ctx context.Context, tenantID, membershipID int64, status, reason string) (*model.TenantMembership, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before tenantMembershipRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ? AND id = ?", tenantID, membershipID).First(&before).Error; err != nil {
			return err
		}
		if before.Status == status {
			return nil
		}
		updates := map[string]any{"status": status, "updated_at": time.Now()}
		if status == "active" && before.JoinedAt == nil {
			updates["joined_at"] = time.Now()
		}
		if err := tx.Model(&tenantMembershipRow{}).Where("tenant_id = ? AND id = ?", tenantID, membershipID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Table("users").Where("id = ?", before.UserID).UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
			return err
		}
		beforeSnapshot := map[string]any{"membership_id": before.ID, "tenant_id": before.TenantID, "user_id": before.UserID, "status": before.Status}
		afterSnapshot := map[string]any{"membership_id": before.ID, "tenant_id": before.TenantID, "user_id": before.UserID, "status": status, "reason": reason}
		return createPlatformResourceAudit(tx, ctx, "tenant.membership.status.update", "tenant_membership", membershipID, tenantID, beforeSnapshot, afterSnapshot)
	})
	if err != nil {
		return nil, fmt.Errorf("update tenant membership status: %w", err)
	}
	return r.GetTenantMembership(ctx, tenantID, membershipID)
}

func (r *TenantRepository) MembershipHasRole(ctx context.Context, membershipID int64, roleKey string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("tenant_membership_roles assignment").
		Joins("JOIN roles role ON role.id = assignment.role_id").
		Where("assignment.membership_id = ? AND assignment.revoked_at IS NULL AND role.role_key = ?", membershipID, roleKey).
		Count(&count).Error
	return count > 0, err
}

func (r *TenantRepository) ListUserMemberships(ctx context.Context, userID int64) ([]model.TenantMembership, error) {
	var rows []membershipTenantRow
	if err := r.db.WithContext(ctx).
		Table("tenant_memberships membership").
		Select("membership.id, membership.tenant_id, membership.user_id, membership.status, membership.joined_at, tenant.tenant_key, tenant.slug, tenant.name AS tenant_name, tenant.status AS tenant_status, tenant.timezone, tenant.locale, tenant.is_default").
		Joins("JOIN tenants tenant ON tenant.id = membership.tenant_id").
		Where("membership.user_id = ?", userID).
		Order("tenant.is_default DESC, tenant.name ASC, tenant.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	memberships := make([]model.TenantMembership, len(rows))
	for i := range rows {
		memberships[i] = rows[i].toDomain()
		roles, err := r.membershipRoles(ctx, rows[i].ID)
		if err != nil {
			return nil, err
		}
		memberships[i].Roles = roles
	}
	return memberships, nil
}

func (r *TenantRepository) GetActiveMembership(ctx context.Context, userID, tenantID int64) (*model.TenantMembership, error) {
	var row membershipTenantRow
	err := r.db.WithContext(ctx).
		Table("tenant_memberships membership").
		Select("membership.id, membership.tenant_id, membership.user_id, membership.status, membership.joined_at, tenant.tenant_key, tenant.slug, tenant.name AS tenant_name, tenant.status AS tenant_status, tenant.timezone, tenant.locale, tenant.is_default").
		Joins("JOIN tenants tenant ON tenant.id = membership.tenant_id").
		Where("membership.user_id = ? AND membership.tenant_id = ? AND membership.status = 'active' AND tenant.status = 'active'", userID, tenantID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	membership := row.toDomain()
	membership.Roles, err = r.membershipRoles(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return &membership, nil
}

func (r *TenantRepository) LoadTenantPrincipal(ctx context.Context, userID, tenantID int64, clientApp string) (*model.Principal, error) {
	membership, err := r.GetActiveMembership(ctx, userID, tenantID)
	if err != nil || membership == nil {
		return nil, err
	}
	var user userRow
	if err := r.db.WithContext(ctx).Where("id = ? AND status = ?", userID, model.UserStatusActive).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	permissions, err := r.membershipPermissions(ctx, membership.ID)
	if err != nil {
		return nil, err
	}
	scopes, err := r.membershipScopes(ctx, membership.ID)
	if err != nil {
		return nil, err
	}
	return &model.Principal{
		UserID: user.ID, Username: user.Username, AccountType: user.AccountType,
		Roles: membership.Roles, Permissions: permissions, DataScopes: scopes,
		TokenVersion: user.TokenVersion, Email: user.Email, LegacyRole: user.Role,
		TenantID: tenantID, MembershipID: membership.ID, ClientApp: clientApp,
	}, nil
}

func (r *TenantRepository) EnsureMembership(ctx context.Context, userID, tenantID int64, status string) (*model.TenantMembership, error) {
	if userID <= 0 || tenantID <= 0 {
		return nil, fmt.Errorf("valid user and tenant are required")
	}
	if status == "" {
		status = "active"
	}
	row := tenantMembershipRow{TenantID: tenantID, UserID: userID, Status: status}
	if status == "active" {
		now := time.Now()
		row.JoinedAt = &now
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{"status": status, "updated_at": time.Now()}),
	}).Create(&row).Error; err != nil {
		return nil, err
	}
	return r.GetActiveMembership(ctx, userID, tenantID)
}

func (r *TenantRepository) AssignMembershipRole(ctx context.Context, membershipID int64, roleID uint64, assignedBy *uint64) error {
	row := tenantMembershipRoleRow{MembershipID: membershipID, RoleID: roleID, AssignedBy: assignedBy, AssignedAt: time.Now()}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func (r *TenantRepository) RevokeMembershipRole(ctx context.Context, membershipID int64, roleID uint64) (bool, error) {
	result := r.db.WithContext(ctx).Model(&tenantMembershipRoleRow{}).
		Where("membership_id = ? AND role_id = ? AND revoked_at IS NULL", membershipID, roleID).
		Update("revoked_at", time.Now())
	return result.RowsAffected > 0, result.Error
}

func (r *TenantRepository) CountActiveTenantMembersWithRole(ctx context.Context, tenantID int64, roleID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("tenant_membership_roles assignment").
		Joins("JOIN tenant_memberships membership ON membership.id = assignment.membership_id").
		Where("membership.tenant_id = ? AND membership.status = 'active' AND assignment.role_id = ? AND assignment.revoked_at IS NULL", tenantID, roleID).
		Count(&count).Error
	return count, err
}

func (r *TenantRepository) AssignMembershipDataScope(ctx context.Context, membershipID int64, scopeKey, resourceType string, resourceID uint64, assignedBy *uint64) error {
	row := tenantMembershipScopeWriteRow{
		MembershipID: membershipID, ScopeKey: scopeKey, ResourceType: resourceType,
		ResourceID: resourceID, AssignedBy: assignedBy, AssignedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func (r *TenantRepository) RevokeMembershipDataScope(ctx context.Context, membershipID, scopeID int64) (bool, error) {
	result := r.db.WithContext(ctx).Model(&tenantMembershipScopeWriteRow{}).
		Where("id = ? AND membership_id = ? AND revoked_at IS NULL", scopeID, membershipID).
		Update("revoked_at", time.Now())
	return result.RowsAffected > 0, result.Error
}

func (r *TenantRepository) RevokeTenantDataScope(ctx context.Context, tenantID, scopeID int64) (int64, bool, error) {
	var owner struct {
		MembershipID int64
		UserID       int64
	}
	err := r.db.WithContext(ctx).Table("tenant_membership_data_scopes scope").
		Select("scope.membership_id, membership.user_id").
		Joins("JOIN tenant_memberships membership ON membership.id = scope.membership_id").
		Where("scope.id = ? AND scope.revoked_at IS NULL AND membership.tenant_id = ?", scopeID, tenantID).
		Scan(&owner).Error
	if err != nil || owner.MembershipID == 0 {
		return 0, false, err
	}
	revoked, err := r.RevokeMembershipDataScope(ctx, owner.MembershipID, scopeID)
	return owner.UserID, revoked, err
}

func (r *TenantRepository) membershipRoles(ctx context.Context, membershipID int64) ([]string, error) {
	var roles []string
	err := r.db.WithContext(ctx).
		Table("tenant_membership_roles assignment").
		Joins("JOIN roles role ON role.id = assignment.role_id AND role.scope_type = ?", model.RoleScopeTenant).
		Where("assignment.membership_id = ? AND assignment.revoked_at IS NULL", membershipID).
		Order("role.role_key ASC").
		Pluck("role.role_key", &roles).Error
	return roles, err
}

func (r *TenantRepository) membershipPermissions(ctx context.Context, membershipID int64) ([]string, error) {
	var permissions []string
	err := r.db.WithContext(ctx).
		Table("tenant_membership_roles assignment").
		Joins("JOIN roles role ON role.id = assignment.role_id AND role.scope_type = ?", model.RoleScopeTenant).
		Joins("JOIN role_permissions mapping ON mapping.role_id = role.id").
		Joins("JOIN permissions permission ON permission.id = mapping.permission_id").
		Where("assignment.membership_id = ? AND assignment.revoked_at IS NULL", membershipID).
		Distinct().Order("permission.permission_key ASC").
		Pluck("permission.permission_key", &permissions).Error
	return permissions, err
}

func (r *TenantRepository) membershipScopes(ctx context.Context, membershipID int64) ([]model.ScopeAssignment, error) {
	var rows []tenantMembershipScopeRow
	if err := r.db.WithContext(ctx).
		Where("membership_id = ? AND revoked_at IS NULL", membershipID).
		Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	scopes := make([]model.ScopeAssignment, len(rows))
	for i := range rows {
		scopes[i] = model.ScopeAssignment{ID: rows[i].ID, ScopeKey: rows[i].ScopeKey, ResourceType: rows[i].ResourceType, ResourceID: rows[i].ResourceID, AssignedAt: rows[i].AssignedAt}
	}
	return scopes, nil
}

type tenantRow struct {
	ID              int64 `gorm:"primaryKey"`
	TenantKey       string
	Slug            string
	Name            string
	Status          string
	Timezone        string
	Locale          string
	IsDefault       bool
	MembershipCount int64 `gorm:"->"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (tenantRow) TableName() string { return "tenants" }

func (r tenantRow) toDomain() *model.Tenant {
	return &model.Tenant{ID: r.ID, TenantKey: r.TenantKey, Slug: r.Slug, Name: r.Name, Status: r.Status, Timezone: r.Timezone, Locale: r.Locale, IsDefault: r.IsDefault, MembershipCount: r.MembershipCount, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

type membershipTenantRow struct {
	ID           int64
	TenantID     int64
	UserID       int64
	Username     string
	Email        string
	Status       string
	JoinedAt     *time.Time
	TenantKey    string
	Slug         string
	TenantName   string
	TenantStatus string
	Timezone     string
	Locale       string
	IsDefault    bool
}

func (r membershipTenantRow) toDomain() model.TenantMembership {
	return model.TenantMembership{
		ID: r.ID, TenantID: r.TenantID, UserID: r.UserID, Username: r.Username, Email: r.Email, Status: r.Status, JoinedAt: r.JoinedAt,
		Tenant: model.Tenant{ID: r.TenantID, TenantKey: r.TenantKey, Slug: r.Slug, Name: r.TenantName, Status: r.TenantStatus, Timezone: r.Timezone, Locale: r.Locale, IsDefault: r.IsDefault},
	}
}

type tenantMembershipScopeRow struct {
	ID           int64 `gorm:"primaryKey"`
	MembershipID int64
	ScopeKey     string
	ResourceType string
	ResourceID   int64
	AssignedAt   time.Time
}

func (tenantMembershipScopeRow) TableName() string { return "tenant_membership_data_scopes" }

type tenantMembershipRow struct {
	ID       int64 `gorm:"primaryKey"`
	TenantID int64
	UserID   int64
	Status   string
	JoinedAt *time.Time
}

func (tenantMembershipRow) TableName() string { return "tenant_memberships" }

type tenantMembershipRoleRow struct {
	ID           int64 `gorm:"primaryKey"`
	MembershipID int64
	RoleID       uint64
	AssignedBy   *uint64
	AssignedAt   time.Time
	RevokedAt    *time.Time
}

func (tenantMembershipRoleRow) TableName() string { return "tenant_membership_roles" }

type tenantMembershipScopeWriteRow struct {
	ID           int64 `gorm:"primaryKey"`
	MembershipID int64
	ScopeKey     string
	ResourceType string
	ResourceID   uint64
	AssignedBy   *uint64
	AssignedAt   time.Time
	RevokedAt    *time.Time
}

func (tenantMembershipScopeWriteRow) TableName() string { return "tenant_membership_data_scopes" }
