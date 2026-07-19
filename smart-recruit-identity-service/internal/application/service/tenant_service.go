package service

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	commonsquota "smart-recruit-commons/quota"
	"smart-recruit-identity-service/internal/domain/model"
	"smart-recruit-identity-service/internal/domain/repository"
	platformmetadata "smart-recruit-platform-go/metadata"
)

var (
	ErrTenantInvalid     = errors.New("invalid tenant")
	ErrLastTenantAdmin   = errors.New("企业必须至少保留一名有效招聘管理员")
	ErrTenantMemberQuota = errors.New("已达到当前套餐的有效成员上限")
	tenantSlugPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)
	tenantLocalePattern  = regexp.MustCompile(`^[a-z]{2,3}(?:-[A-Z]{2})?$`)
)

type TenantService struct {
	repo  repository.TenantRepository
	authz repository.AuthzRepository
}

func NewTenantService(repo repository.TenantRepository, authz repository.AuthzRepository) *TenantService {
	return &TenantService{repo: repo, authz: authz}
}

func (s *TenantService) Create(ctx context.Context, slug, name, timezone, locale string) (*model.Tenant, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformTenantManage); err != nil {
		return nil, err
	}
	slug = strings.ToLower(strings.TrimSpace(slug))
	name = strings.TrimSpace(name)
	if s.repo == nil || !tenantSlugPattern.MatchString(slug) || name == "" || len(name) > 128 {
		return nil, ErrTenantInvalid
	}
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	if locale == "" {
		locale = "zh-CN"
	}
	if _, err := time.LoadLocation(timezone); err != nil || !tenantLocalePattern.MatchString(locale) {
		return nil, ErrTenantInvalid
	}
	tenant := &model.Tenant{TenantKey: uuid.NewString(), Slug: slug, Name: name, Status: "active", Timezone: timezone, Locale: locale}
	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *TenantService) List(ctx context.Context, page, pageSize int32, keyword, status string) ([]model.Tenant, int64, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformTenantRead); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	status = strings.TrimSpace(status)
	if status != "" && status != "active" && status != "suspended" && status != "disabled" {
		return nil, 0, ErrTenantInvalid
	}
	return s.repo.List(ctx, int((page-1)*pageSize), int(pageSize), strings.TrimSpace(keyword), status)
}

func (s *TenantService) Get(ctx context.Context, tenantID int64) (*model.Tenant, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformTenantRead); err != nil {
		return nil, err
	}
	if tenantID <= 0 || s.repo == nil {
		return nil, ErrTenantInvalid
	}
	tenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantInvalid
	}
	return tenant, nil
}

func (s *TenantService) UpdateStatus(ctx context.Context, tenantID int64, status, reason string) (*model.Tenant, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformTenantManage); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if tenantID <= 0 || (status != "active" && status != "suspended" && status != "disabled") || reason == "" || len(reason) > 500 {
		return nil, ErrTenantInvalid
	}
	tenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant == nil || tenant.IsDefault {
		return nil, ErrTenantInvalid
	}
	if tenant.Status == status {
		return tenant, nil
	}
	return s.repo.UpdateStatus(ctx, tenantID, status, reason)
}

func (s *TenantService) ListMemberships(ctx context.Context, tenantID int64, page, pageSize int32) ([]model.TenantMembership, int64, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformTenantRead); err != nil {
		return nil, 0, err
	}
	if tenantID <= 0 {
		return nil, 0, ErrTenantInvalid
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListTenantMemberships(ctx, tenantID, int((page-1)*pageSize), int(pageSize))
}

func (s *TenantService) UpdateMembershipStatus(ctx context.Context, tenantID, membershipID int64, status, reason string) (*model.TenantMembership, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformMemberManage); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if tenantID <= 0 || membershipID <= 0 || (status != "active" && status != "suspended") || reason == "" || len(reason) > 500 || s.repo == nil {
		return nil, ErrTenantInvalid
	}
	membership, err := s.repo.GetTenantMembership(ctx, tenantID, membershipID)
	if err != nil || membership == nil {
		if err != nil {
			return nil, err
		}
		return nil, ErrTenantInvalid
	}
	if membership.Status == status {
		return membership, nil
	}
	if status == "active" {
		if err := s.repo.CheckTenantQuota(ctx, tenantID, "members.max", 1); err != nil {
			if errors.Is(err, commonsquota.ErrLimitExceeded) {
				return nil, ErrTenantMemberQuota
			}
			return nil, err
		}
	}
	if status == "suspended" {
		hasAdminRole, err := s.repo.MembershipHasRole(ctx, membershipID, model.RoleRecruitingAdmin)
		if err != nil {
			return nil, err
		}
		if hasAdminRole {
			role, err := s.authz.GetRoleByKey(ctx, model.RoleRecruitingAdmin)
			if err != nil {
				return nil, err
			}
			if role == nil {
				return nil, ErrTenantInvalid
			}
			count, err := s.repo.CountActiveTenantMembersWithRole(ctx, tenantID, role.ID)
			if err != nil {
				return nil, err
			}
			if count <= 1 {
				return nil, ErrLastTenantAdmin
			}
		}
	}
	return s.repo.UpdateTenantMembershipStatus(ctx, tenantID, membershipID, status, reason)
}

func (s *TenantService) Dashboard(ctx context.Context) (model.PlatformDashboard, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformDashboardRead); err != nil {
		return model.PlatformDashboard{}, err
	}
	if s.repo == nil {
		return model.PlatformDashboard{}, ErrTenantInvalid
	}
	return s.repo.GetPlatformDashboard(ctx)
}

func (s *TenantService) QueryAuditLogs(ctx context.Context, filter model.PlatformAuditFilter, page, pageSize int32) ([]model.PlatformAuditLog, int64, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformAuditRead); err != nil {
		return nil, 0, err
	}
	if filter.TenantID < 0 || filter.ActorUserID < 0 || filter.Action != strings.TrimSpace(filter.Action) || (filter.StartTime != nil && filter.EndTime != nil && filter.StartTime.After(*filter.EndTime)) {
		return nil, 0, ErrTenantInvalid
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.QueryPlatformAuditLogs(ctx, filter, int((page-1)*pageSize), int(pageSize))
}

func (s *TenantService) ListPlans(ctx context.Context, status string) ([]model.PlatformPlan, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformPlanRead); err != nil {
		return nil, err
	}
	status = strings.TrimSpace(status)
	if status != "" && status != "active" && status != "retired" {
		return nil, ErrTenantInvalid
	}
	return s.repo.ListPlatformPlans(ctx, status)
}

func (s *TenantService) SavePlanVersion(ctx context.Context, planID, versionID int64, changeNote string, entitlements []model.PlatformEntitlement) (*model.PlatformPlanVersion, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformPlanManage); err != nil {
		return nil, err
	}
	changeNote = strings.TrimSpace(changeNote)
	if planID <= 0 || versionID < 0 || changeNote == "" || len(changeNote) > 500 || len(entitlements) == 0 || len(entitlements) > 50 {
		return nil, ErrTenantInvalid
	}
	allowedKeys := map[string]bool{"members.max": true, "jobs.published.max": true, "applications.monthly.max": true, "resumes.storage.max": true}
	seen := map[string]bool{}
	for i := range entitlements {
		item := &entitlements[i]
		item.Key = strings.TrimSpace(item.Key)
		item.ValueType = strings.TrimSpace(item.ValueType)
		item.EnforcementMode = strings.TrimSpace(item.EnforcementMode)
		if !allowedKeys[item.Key] || seen[item.Key] || item.ValueType != "integer" || (item.EnforcementMode != "hard" && item.EnforcementMode != "soft" && item.EnforcementMode != "observe") || !json.Valid([]byte(item.ValueJSON)) {
			return nil, ErrTenantInvalid
		}
		var value int64
		if err := json.Unmarshal([]byte(item.ValueJSON), &value); err != nil || value <= 0 {
			return nil, ErrTenantInvalid
		}
		seen[item.Key] = true
	}
	return s.repo.SavePlatformPlanVersion(ctx, planID, versionID, changeNote, entitlements)
}

func (s *TenantService) PublishPlanVersion(ctx context.Context, planID, versionID int64, effectiveAt time.Time, reason string) (*model.PlatformPlanVersion, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformPlanPublish); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if planID <= 0 || versionID <= 0 || effectiveAt.IsZero() || reason == "" || len(reason) > 500 {
		return nil, ErrTenantInvalid
	}
	return s.repo.PublishPlatformPlanVersion(ctx, planID, versionID, effectiveAt, reason)
}

func (s *TenantService) GetSubscription(ctx context.Context, tenantID int64) (*model.TenantSubscription, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformTenantRead); err != nil {
		return nil, err
	}
	if tenantID <= 0 {
		return nil, ErrTenantInvalid
	}
	return s.repo.GetTenantSubscription(ctx, tenantID)
}

func (s *TenantService) UpdateSubscription(ctx context.Context, tenantID, planVersionID int64, startsAt time.Time, endsAt *time.Time, reason string) (*model.TenantSubscription, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformSubscriptionManage); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if tenantID <= 0 || planVersionID <= 0 || startsAt.IsZero() || startsAt.After(time.Now()) || (endsAt != nil && !endsAt.After(startsAt)) || reason == "" || len(reason) > 500 {
		return nil, ErrTenantInvalid
	}
	return s.repo.UpdateTenantSubscription(ctx, tenantID, planVersionID, startsAt, endsAt, reason)
}

func (s *TenantService) UpdateEntitlementOverride(ctx context.Context, tenantID int64, entitlement model.PlatformEntitlement, expiresAt *time.Time, reason string) (*model.TenantSubscription, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformPlanManage); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	allowedKeys := map[string]bool{"members.max": true, "jobs.published.max": true, "applications.monthly.max": true, "resumes.storage.max": true}
	var value int64
	if tenantID <= 0 || !allowedKeys[entitlement.Key] || entitlement.ValueType != "integer" || json.Unmarshal([]byte(entitlement.ValueJSON), &value) != nil || value <= 0 || (expiresAt != nil && !expiresAt.After(time.Now())) || reason == "" || len(reason) > 500 {
		return nil, ErrTenantInvalid
	}
	entitlement.Source = "override"
	return s.repo.UpdateTenantEntitlementOverride(ctx, tenantID, entitlement, expiresAt, reason)
}

func (s *TenantService) GetUsage(ctx context.Context, tenantID int64) ([]model.TenantUsageMetric, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformUsageRead); err != nil {
		return nil, err
	}
	if tenantID <= 0 {
		return nil, ErrTenantInvalid
	}
	return s.repo.GetTenantUsage(ctx, tenantID)
}

func (s *TenantService) ListAlerts(ctx context.Context, filter model.QuotaAlertFilter, page, pageSize int32) ([]model.QuotaAlert, int64, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformAlertRead); err != nil {
		return nil, 0, err
	}
	filter.Status = strings.TrimSpace(filter.Status)
	filter.MetricKey = strings.TrimSpace(filter.MetricKey)
	if filter.TenantID < 0 || (filter.Status != "" && filter.Status != "open" && filter.Status != "acknowledged" && filter.Status != "resolved") {
		return nil, 0, ErrTenantInvalid
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListQuotaAlerts(ctx, filter, int((page-1)*pageSize), int(pageSize))
}

func (s *TenantService) UpdateAlert(ctx context.Context, alertID int64, status string, assigneeUserID int64, resolutionNote string) (*model.QuotaAlert, error) {
	if err := s.authorizePlatform(ctx, model.PermPlatformAlertManage); err != nil {
		return nil, err
	}
	status = strings.TrimSpace(status)
	resolutionNote = strings.TrimSpace(resolutionNote)
	if alertID <= 0 || assigneeUserID < 0 || (status != "acknowledged" && status != "resolved") || (status == "resolved" && resolutionNote == "") || len(resolutionNote) > 500 {
		return nil, ErrTenantInvalid
	}
	return s.repo.UpdateQuotaAlert(ctx, alertID, status, assigneeUserID, resolutionNote)
}

func (s *TenantService) authorizePlatform(ctx context.Context, permission string) error {
	if s.authz == nil || platformmetadata.GetAuthClientApp(ctx) != "platform" || platformmetadata.GetAuthTenantID(ctx) != 0 {
		return ErrInvalidCredentials
	}
	principal, err := s.authz.LoadPlatformPrincipal(ctx, uint64(platformmetadata.GetAuthUserID(ctx)))
	if err != nil || principal == nil {
		return ErrInvalidCredentials
	}
	for _, granted := range principal.Permissions {
		if granted == permission {
			return nil
		}
	}
	return ErrInvalidCredentials
}
