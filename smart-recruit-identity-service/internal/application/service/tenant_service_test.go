package service

import (
	"context"
	"errors"
	"testing"

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
	if _, err := service.UpdateStatus(ctx, 7, "archived"); !errors.Is(err, ErrTenantInvalid) {
		t.Fatalf("UpdateStatus error = %v, want ErrTenantInvalid", err)
	}
}

type tenantRepositoryFake struct {
	lastOffset  int
	lastLimit   int
	lastKeyword string
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
func (r *tenantRepositoryFake) UpdateStatus(_ context.Context, tenantID int64, status string) (*model.Tenant, error) {
	return &model.Tenant{ID: tenantID, Status: status}, nil
}
func (r *tenantRepositoryFake) ListTenantMemberships(context.Context, int64, int, int) ([]model.TenantMembership, int64, error) {
	return nil, 0, nil
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
	return 0, nil
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
