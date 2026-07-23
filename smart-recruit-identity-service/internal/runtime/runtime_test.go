package runtime

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestRuntimeRegistersIdentityGRPCServices(t *testing.T) {
	runtime, err := New(Deps{
		Auth:   fakeAuthAPI{},
		Admin:  fakeAdminAPI{},
		Audit:  fakeAuditAPI{},
		Tenant: fakeTenantAPI{},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)
	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC returned error: %v", err)
	}
	services := server.GetServiceInfo()
	if _, ok := services[pb.AuthService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("missing registered service %s", pb.AuthService_ServiceDesc.ServiceName)
	}
	if _, ok := services[pb.AdminService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("missing registered service %s", pb.AdminService_ServiceDesc.ServiceName)
	}
	if _, ok := services[pb.PlatformTenantService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("missing registered service %s", pb.PlatformTenantService_ServiceDesc.ServiceName)
	}
}

func TestRuntimeRequiresDependencies(t *testing.T) {
	if _, err := New(Deps{}); err == nil {
		t.Fatal("expected missing deps error")
	}
}

type fakeAuthAPI struct{}

func (fakeAuthAPI) Register(context.Context, *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return &pb.RegisterResponse{Code: errs.OK}, nil
}

func (fakeAuthAPI) Login(context.Context, *pb.LoginRequest) (*pb.LoginResponse, error) {
	return &pb.LoginResponse{Code: errs.OK}, nil
}

func (fakeAuthAPI) SwitchTenant(context.Context, *pb.SwitchTenantRequest) (*pb.LoginResponse, error) {
	return &pb.LoginResponse{Code: errs.OK}, nil
}

func (fakeAuthAPI) RefreshToken(context.Context, *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return &pb.RefreshTokenResponse{Code: errs.OK}, nil
}

func (fakeAuthAPI) RevokeRefreshToken(context.Context, *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeAuthAPI) RecordAuthDecision(context.Context, *pb.AuthAuditRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeAuthAPI) GetPrincipal(context.Context, *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error) {
	return &pb.GetPrincipalResponse{Code: errs.OK}, nil
}

func (fakeAuthAPI) AuthorizeInternal(context.Context, *pb.AuthorizeInternalRequest) (*pb.AuthorizeInternalResponse, error) {
	return &pb.AuthorizeInternalResponse{Code: errs.OK, Allowed: true}, nil
}

func (fakeAuthAPI) UpdateEmail(context.Context, *pb.UpdateEmailRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

type fakeAdminAPI struct{}

func (fakeAdminAPI) ListRoles(context.Context, *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	return &pb.ListRolesResponse{Code: errs.OK}, nil
}

func (fakeAdminAPI) ListPermissions(context.Context, *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	return &pb.ListPermissionsResponse{Code: errs.OK}, nil
}

func (fakeAdminAPI) GetUserRoles(context.Context, *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	return &pb.GetUserRolesResponse{Code: errs.OK}, nil
}

func (fakeAdminAPI) AssignUserRole(context.Context, *pb.AssignUserRoleRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeAdminAPI) RevokeUserRole(context.Context, *pb.RevokeUserRoleRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeAdminAPI) AssignDataScope(context.Context, *pb.AssignDataScopeRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeAdminAPI) RevokeDataScope(context.Context, *pb.RevokeDataScopeRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (fakeAdminAPI) ListStaffUsers(context.Context, *pb.ListStaffUsersRequest) (*pb.ListStaffUsersResponse, error) {
	return &pb.ListStaffUsersResponse{Code: errs.OK}, nil
}

func (fakeAdminAPI) CreateStaffUser(context.Context, *pb.CreateStaffUserRequest) (*pb.CreateStaffUserResponse, error) {
	return &pb.CreateStaffUserResponse{Code: errs.OK}, nil
}
func (fakeAdminAPI) ListPlatformUsers(context.Context, *pb.ListPlatformUsersRequest) (*pb.ListPlatformUsersResponse, error) {
	return &pb.ListPlatformUsersResponse{Code: errs.OK}, nil
}
func (fakeAdminAPI) CreatePlatformUser(context.Context, *pb.CreatePlatformUserRequest) (*pb.CreatePlatformUserResponse, error) {
	return &pb.CreatePlatformUserResponse{Code: errs.OK}, nil
}
func (fakeAdminAPI) UpdatePlatformUser(context.Context, *pb.UpdatePlatformUserRequest) (*pb.PlatformUserResponse, error) {
	return &pb.PlatformUserResponse{Code: errs.OK}, nil
}

type fakeAuditAPI struct{}

func (fakeAuditAPI) QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error) {
	return &pb.QueryAuthAuditLogsResponse{Code: errs.OK}, nil
}

type fakeTenantAPI struct{}

func (fakeTenantAPI) CreateTenant(context.Context, *pb.CreateTenantRequest) (*pb.TenantResponse, error) {
	return &pb.TenantResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) ListTenants(context.Context, *pb.ListTenantsRequest) (*pb.ListTenantsResponse, error) {
	return &pb.ListTenantsResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) GetTenant(context.Context, *pb.GetTenantRequest) (*pb.TenantResponse, error) {
	return &pb.TenantResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) UpdateTenantStatus(context.Context, *pb.UpdateTenantStatusRequest) (*pb.TenantResponse, error) {
	return &pb.TenantResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) ListTenantMemberships(context.Context, *pb.ListTenantMembershipsRequest) (*pb.ListTenantMembershipsResponse, error) {
	return &pb.ListTenantMembershipsResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) UpdateTenantMembershipStatus(context.Context, *pb.UpdateTenantMembershipStatusRequest) (*pb.TenantMembershipResponse, error) {
	return &pb.TenantMembershipResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) GetPlatformDashboard(context.Context, *pb.GetPlatformDashboardRequest) (*pb.GetPlatformDashboardResponse, error) {
	return &pb.GetPlatformDashboardResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) QueryPlatformAuditLogs(context.Context, *pb.QueryPlatformAuditLogsRequest) (*pb.QueryPlatformAuditLogsResponse, error) {
	return &pb.QueryPlatformAuditLogsResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) ListPlatformPlans(context.Context, *pb.ListPlatformPlansRequest) (*pb.ListPlatformPlansResponse, error) {
	return &pb.ListPlatformPlansResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) SavePlatformPlanVersion(context.Context, *pb.SavePlatformPlanVersionRequest) (*pb.PlatformPlanVersionResponse, error) {
	return &pb.PlatformPlanVersionResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) PublishPlatformPlanVersion(context.Context, *pb.PublishPlatformPlanVersionRequest) (*pb.PlatformPlanVersionResponse, error) {
	return &pb.PlatformPlanVersionResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) GetTenantSubscription(context.Context, *pb.GetTenantSubscriptionRequest) (*pb.TenantSubscriptionResponse, error) {
	return &pb.TenantSubscriptionResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) UpdateTenantSubscription(context.Context, *pb.UpdateTenantSubscriptionRequest) (*pb.TenantSubscriptionResponse, error) {
	return &pb.TenantSubscriptionResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) UpdateTenantEntitlementOverride(context.Context, *pb.UpdateTenantEntitlementOverrideRequest) (*pb.TenantSubscriptionResponse, error) {
	return &pb.TenantSubscriptionResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) GetTenantUsage(context.Context, *pb.GetTenantUsageRequest) (*pb.GetTenantUsageResponse, error) {
	return &pb.GetTenantUsageResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) ListQuotaAlerts(context.Context, *pb.ListQuotaAlertsRequest) (*pb.ListQuotaAlertsResponse, error) {
	return &pb.ListQuotaAlertsResponse{Code: errs.OK}, nil
}
func (fakeTenantAPI) UpdateQuotaAlert(context.Context, *pb.UpdateQuotaAlertRequest) (*pb.QuotaAlertResponse, error) {
	return &pb.QuotaAlertResponse{Code: errs.OK}, nil
}
