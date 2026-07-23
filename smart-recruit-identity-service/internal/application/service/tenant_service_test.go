package service

import (
	"context"
	"errors"
	"testing"
	"time"

	commonsquota "smart-recruit-commons/quota"
	"smart-recruit-identity-service/internal/domain/model"
	platformmetadata "smart-recruit-platform-go/metadata"
)

func TestTenantServiceRequiresPlatformApplication(t *testing.T) {
	service := NewTenantService(&tenantRepositoryFake{}, newMemoryAuthz())
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{name: "missing principal", ctx: context.Background()},
		{name: "staff workspace", ctx: platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{UserID: 1, AccountType: "staff", ClientApp: "hr", TenantID: 7, MembershipID: 8})},
		{name: "platform principal bound to tenant", ctx: platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{UserID: 1, AccountType: "platform", ClientApp: "platform", TenantID: 7})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := service.List(test.ctx, 1, 20, "", ""); !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("List error = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

func TestTenantServiceCreatesNormalizedTenantAndBoundsPagination(t *testing.T) {
	repo := &tenantRepositoryFake{}
	service := NewTenantService(repo, newMemoryAuthz())
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{UserID: 12, AccountType: "platform", ClientApp: "platform"})

	tenant, err := service.Create(ctx, " Acme-China ", " Acme 中国 ", "", "")
	if err != nil {
		t.Fatalf("Create returned %v", err)
	}
	if tenant.Slug != "acme-china" || tenant.Name != "Acme 中国" || tenant.Timezone != "Asia/Shanghai" || tenant.Locale != "zh-CN" {
		t.Fatalf("unexpected tenant: %+v", tenant)
	}
	if tenant.ID == 0 || tenant.TenantKey == "" {
		t.Fatalf("tenant identity was not generated: %+v", tenant)
	}

	if _, _, err := service.List(ctx, 0, 999, " acme ", "active"); err != nil {
		t.Fatalf("List returned %v", err)
	}
	if repo.lastOffset != 0 || repo.lastLimit != 20 || repo.lastKeyword != "acme" {
		t.Fatalf("pagination was not normalized: offset=%d limit=%d keyword=%q", repo.lastOffset, repo.lastLimit, repo.lastKeyword)
	}
}

func TestTenantServiceRejectsInvalidLifecycleInput(t *testing.T) {
	service := NewTenantService(&tenantRepositoryFake{}, newMemoryAuthz())
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{UserID: 12, AccountType: "platform", ClientApp: "platform"})
	if _, err := service.Create(ctx, "INVALID SLUG", "Acme", "", ""); !errors.Is(err, ErrTenantInvalid) {
		t.Fatalf("Create error = %v, want ErrTenantInvalid", err)
	}
	if _, err := service.UpdateStatus(ctx, 7, "archived", "test"); !errors.Is(err, ErrTenantInvalid) {
		t.Fatalf("UpdateStatus error = %v, want ErrTenantInvalid", err)
	}
}

func TestTenantServiceUsesPermissionSpecificPlatformAuthorization(t *testing.T) {
	authz := newMemoryAuthz()
	authz.platformPerms = []string{model.PermPlatformTenantRead}
	service := NewTenantService(&tenantRepositoryFake{}, authz)
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{UserID: 12, AccountType: "platform", ClientApp: "platform"})
	if _, _, err := service.List(ctx, 1, 20, "", ""); err != nil {
		t.Fatalf("List returned %v for tenant read permission", err)
	}
	if _, err := service.Create(ctx, "acme-test", "Acme", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Create error = %v, want permission denial", err)
	}
}

func TestTenantServiceProtectsLastActiveTenantAdmin(t *testing.T) {
	repo := &tenantRepositoryFake{
		membership: &model.TenantMembership{ID: 9, TenantID: 7, UserID: 22, Status: "active", Roles: []string{model.RoleRecruitingAdmin}},
		hasRole:    true, activeAdminCount: 1,
	}
	service := NewTenantService(repo, newMemoryAuthz())
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{UserID: 12, AccountType: "platform", ClientApp: "platform"})
	if _, err := service.UpdateMembershipStatus(ctx, 7, 9, "suspended", "security response"); !errors.Is(err, ErrLastTenantAdmin) {
		t.Fatalf("UpdateMembershipStatus error = %v, want ErrLastTenantAdmin", err)
	}
	if repo.updateMembershipCalled {
		t.Fatal("last active tenant admin must not be suspended")
	}
}

func TestTenantServiceEnforcesMemberQuotaWhenRestoringMembership(t *testing.T) {
	repo := &tenantRepositoryFake{
		membership: &model.TenantMembership{ID: 9, TenantID: 7, UserID: 22, Status: "suspended"},
		quotaErr:   commonsquota.ErrLimitExceeded,
	}
	service := NewTenantService(repo, newMemoryAuthz())
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{UserID: 12, AccountType: "platform", ClientApp: "platform"})
	if _, err := service.UpdateMembershipStatus(ctx, 7, 9, "active", "approved restoration"); !errors.Is(err, ErrTenantMemberQuota) {
		t.Fatalf("UpdateMembershipStatus error = %v, want ErrTenantMemberQuota", err)
	}
	if repo.updateMembershipCalled {
		t.Fatal("membership restoration must not bypass the active member quota")
	}
}

func TestTenantServiceSeparatesPlanManageAndPublishPermissions(t *testing.T) {
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{UserID: 12, AccountType: "platform", ClientApp: "platform"})
	entitlements := []model.PlatformEntitlement{{Key: "members.max", ValueType: "integer", ValueJSON: "100", EnforcementMode: "hard"}}

	manageAuthz := newMemoryAuthz()
	manageAuthz.platformPerms = []string{model.PermPlatformPlanManage}
	manageService := NewTenantService(&tenantRepositoryFake{}, manageAuthz)
	if _, err := manageService.SavePlanVersion(ctx, 1, 0, "initial draft", entitlements); err != nil {
		t.Fatalf("SavePlanVersion returned %v for plan manage permission", err)
	}
	if _, err := manageService.PublishPlanVersion(ctx, 1, 2, time.Now(), "release"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("PublishPlanVersion error = %v, want permission denial", err)
	}

	publishAuthz := newMemoryAuthz()
	publishAuthz.platformPerms = []string{model.PermPlatformPlanPublish}
	publishService := NewTenantService(&tenantRepositoryFake{}, publishAuthz)
	if _, err := publishService.PublishPlanVersion(ctx, 1, 2, time.Now(), "release"); err != nil {
		t.Fatalf("PublishPlanVersion returned %v for plan publish permission", err)
	}
	if _, err := publishService.SavePlanVersion(ctx, 1, 0, "initial draft", entitlements); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("SavePlanVersion error = %v, want permission denial", err)
	}
}

func TestTenantServiceValidatesAndForwardsEntitlementOverride(t *testing.T) {
	repo := &tenantRepositoryFake{}
	authz := newMemoryAuthz()
	authz.platformPerms = []string{model.PermPlatformPlanManage}
	service := NewTenantService(repo, authz)
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{UserID: 12, AccountType: "platform", ClientApp: "platform"})

	if _, err := service.UpdateEntitlementOverride(ctx, 7, model.PlatformEntitlement{Key: "unsupported", ValueType: "integer", ValueJSON: "10"}, nil, "approved"); !errors.Is(err, ErrTenantInvalid) {
		t.Fatalf("UpdateEntitlementOverride error = %v, want ErrTenantInvalid", err)
	}
	if repo.overrideCalled {
		t.Fatal("invalid override must not reach repository")
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if _, err := service.UpdateEntitlementOverride(ctx, 7, model.PlatformEntitlement{Key: "members.max", ValueType: "integer", ValueJSON: "150"}, &expiresAt, "temporary expansion"); err != nil {
		t.Fatalf("UpdateEntitlementOverride returned %v", err)
	}
	if !repo.overrideCalled || repo.overrideEntitlement.Source != "override" || repo.overrideEntitlement.ValueJSON != "150" {
		t.Fatalf("override was not normalized and forwarded: %+v", repo.overrideEntitlement)
	}
}

func TestTenantServiceRejectsFutureSubscriptionStartInInitialPhases(t *testing.T) {
	authz := newMemoryAuthz()
	authz.platformPerms = []string{model.PermPlatformSubscriptionManage}
	service := NewTenantService(&tenantRepositoryFake{}, authz)
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{UserID: 12, AccountType: "platform", ClientApp: "platform"})
	if _, err := service.UpdateSubscription(ctx, 7, 3, time.Now().Add(time.Hour), nil, "scheduled migration"); !errors.Is(err, ErrTenantInvalid) {
		t.Fatalf("UpdateSubscription error = %v, want ErrTenantInvalid", err)
	}
}

type tenantRepositoryFake struct {
	lastOffset             int
	lastLimit              int
	lastKeyword            string
	membership             *model.TenantMembership
	hasRole                bool
	activeAdminCount       int64
	updateMembershipCalled bool
	overrideCalled         bool
	overrideEntitlement    model.PlatformEntitlement
	quotaErr               error
}

func (r *tenantRepositoryFake) GetByID(context.Context, int64) (*model.Tenant, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) GetDefault(context.Context) (*model.Tenant, error) { return nil, nil }
func (r *tenantRepositoryFake) Create(_ context.Context, tenant *model.Tenant) error {
	tenant.ID = 42
	return nil
}
func (r *tenantRepositoryFake) List(_ context.Context, offset, limit int, keyword, _ string) ([]model.Tenant, int64, error) {
	r.lastOffset, r.lastLimit, r.lastKeyword = offset, limit, keyword
	return nil, 0, nil
}
func (r *tenantRepositoryFake) UpdateStatus(_ context.Context, tenantID int64, status, _ string) (*model.Tenant, error) {
	return &model.Tenant{ID: tenantID, Status: status}, nil
}
func (r *tenantRepositoryFake) GetPlatformDashboard(context.Context) (model.PlatformDashboard, error) {
	return model.PlatformDashboard{}, nil
}
func (r *tenantRepositoryFake) QueryPlatformAuditLogs(context.Context, model.PlatformAuditFilter, int, int) ([]model.PlatformAuditLog, int64, error) {
	return nil, 0, nil
}
func (r *tenantRepositoryFake) ListPlatformPlans(context.Context, string) ([]model.PlatformPlan, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) SavePlatformPlanVersion(context.Context, int64, int64, string, []model.PlatformEntitlement) (*model.PlatformPlanVersion, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) PublishPlatformPlanVersion(context.Context, int64, int64, time.Time, string) (*model.PlatformPlanVersion, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) GetTenantSubscription(context.Context, int64) (*model.TenantSubscription, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) UpdateTenantSubscription(context.Context, int64, int64, time.Time, *time.Time, string) (*model.TenantSubscription, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) UpdateTenantEntitlementOverride(_ context.Context, _ int64, entitlement model.PlatformEntitlement, _ *time.Time, _ string) (*model.TenantSubscription, error) {
	r.overrideCalled = true
	r.overrideEntitlement = entitlement
	return &model.TenantSubscription{}, nil
}
func (r *tenantRepositoryFake) GetTenantUsage(context.Context, int64) ([]model.TenantUsageMetric, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) ListQuotaAlerts(context.Context, model.QuotaAlertFilter, int, int) ([]model.QuotaAlert, int64, error) {
	return nil, 0, nil
}
func (r *tenantRepositoryFake) UpdateQuotaAlert(context.Context, int64, string, int64, string) (*model.QuotaAlert, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) CheckTenantQuota(context.Context, int64, string, int64) error {
	return r.quotaErr
}
func (r *tenantRepositoryFake) ListTenantMemberships(context.Context, int64, int, int) ([]model.TenantMembership, int64, error) {
	return nil, 0, nil
}
func (r *tenantRepositoryFake) GetTenantMembership(context.Context, int64, int64) (*model.TenantMembership, error) {
	return r.membership, nil
}
func (r *tenantRepositoryFake) UpdateTenantMembershipStatus(context.Context, int64, int64, string, string) (*model.TenantMembership, error) {
	r.updateMembershipCalled = true
	return r.membership, nil
}
func (r *tenantRepositoryFake) MembershipHasRole(context.Context, int64, string) (bool, error) {
	return r.hasRole, nil
}
func (r *tenantRepositoryFake) ListUserMemberships(context.Context, int64) ([]model.TenantMembership, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) GetActiveMembership(context.Context, int64, int64) (*model.TenantMembership, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) LoadTenantPrincipal(context.Context, int64, int64, string) (*model.Principal, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) EnsureMembership(context.Context, int64, int64, string) (*model.TenantMembership, error) {
	return nil, nil
}
func (r *tenantRepositoryFake) AssignMembershipRole(context.Context, int64, uint64, *uint64) error {
	return nil
}
func (r *tenantRepositoryFake) RevokeMembershipRole(context.Context, int64, uint64) (bool, error) {
	return false, nil
}
func (r *tenantRepositoryFake) CountActiveTenantMembersWithRole(context.Context, int64, uint64) (int64, error) {
	return r.activeAdminCount, nil
}
func (r *tenantRepositoryFake) AssignMembershipDataScope(context.Context, int64, string, string, uint64, *uint64) error {
	return nil
}
func (r *tenantRepositoryFake) RevokeMembershipDataScope(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (r *tenantRepositoryFake) RevokeTenantDataScope(context.Context, int64, int64) (int64, bool, error) {
	return 0, false, nil
}
