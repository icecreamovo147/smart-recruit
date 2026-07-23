package runtime

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"smart-recruit-proto/recruitment/pb"
)

const ServiceName = "identity-service"

type AuthAPI interface {
	Register(context.Context, *pb.RegisterRequest) (*pb.RegisterResponse, error)
	Login(context.Context, *pb.LoginRequest) (*pb.LoginResponse, error)
	RefreshToken(context.Context, *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error)
	SwitchTenant(context.Context, *pb.SwitchTenantRequest) (*pb.LoginResponse, error)
	RevokeRefreshToken(context.Context, *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error)
	RecordAuthDecision(context.Context, *pb.AuthAuditRequest) (*pb.CommonResponse, error)
	GetPrincipal(context.Context, *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error)
	AuthorizeInternal(context.Context, *pb.AuthorizeInternalRequest) (*pb.AuthorizeInternalResponse, error)
	UpdateEmail(context.Context, *pb.UpdateEmailRequest) (*pb.CommonResponse, error)
}

type AdminAPI interface {
	ListRoles(context.Context, *pb.ListRolesRequest) (*pb.ListRolesResponse, error)
	ListPermissions(context.Context, *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error)
	GetUserRoles(context.Context, *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error)
	AssignUserRole(context.Context, *pb.AssignUserRoleRequest) (*pb.CommonResponse, error)
	RevokeUserRole(context.Context, *pb.RevokeUserRoleRequest) (*pb.CommonResponse, error)
	AssignDataScope(context.Context, *pb.AssignDataScopeRequest) (*pb.CommonResponse, error)
	RevokeDataScope(context.Context, *pb.RevokeDataScopeRequest) (*pb.CommonResponse, error)
	ListStaffUsers(context.Context, *pb.ListStaffUsersRequest) (*pb.ListStaffUsersResponse, error)
	CreateStaffUser(context.Context, *pb.CreateStaffUserRequest) (*pb.CreateStaffUserResponse, error)
	ListPlatformUsers(context.Context, *pb.ListPlatformUsersRequest) (*pb.ListPlatformUsersResponse, error)
	CreatePlatformUser(context.Context, *pb.CreatePlatformUserRequest) (*pb.CreatePlatformUserResponse, error)
	UpdatePlatformUser(context.Context, *pb.UpdatePlatformUserRequest) (*pb.PlatformUserResponse, error)
}

type AuditAPI interface {
	QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error)
}

type TenantAPI interface {
	CreateTenant(context.Context, *pb.CreateTenantRequest) (*pb.TenantResponse, error)
	ListTenants(context.Context, *pb.ListTenantsRequest) (*pb.ListTenantsResponse, error)
	GetTenant(context.Context, *pb.GetTenantRequest) (*pb.TenantResponse, error)
	UpdateTenantStatus(context.Context, *pb.UpdateTenantStatusRequest) (*pb.TenantResponse, error)
	ListTenantMemberships(context.Context, *pb.ListTenantMembershipsRequest) (*pb.ListTenantMembershipsResponse, error)
	UpdateTenantMembershipStatus(context.Context, *pb.UpdateTenantMembershipStatusRequest) (*pb.TenantMembershipResponse, error)
	GetPlatformDashboard(context.Context, *pb.GetPlatformDashboardRequest) (*pb.GetPlatformDashboardResponse, error)
	QueryPlatformAuditLogs(context.Context, *pb.QueryPlatformAuditLogsRequest) (*pb.QueryPlatformAuditLogsResponse, error)
	ListPlatformPlans(context.Context, *pb.ListPlatformPlansRequest) (*pb.ListPlatformPlansResponse, error)
	SavePlatformPlanVersion(context.Context, *pb.SavePlatformPlanVersionRequest) (*pb.PlatformPlanVersionResponse, error)
	PublishPlatformPlanVersion(context.Context, *pb.PublishPlatformPlanVersionRequest) (*pb.PlatformPlanVersionResponse, error)
	GetTenantSubscription(context.Context, *pb.GetTenantSubscriptionRequest) (*pb.TenantSubscriptionResponse, error)
	UpdateTenantSubscription(context.Context, *pb.UpdateTenantSubscriptionRequest) (*pb.TenantSubscriptionResponse, error)
	UpdateTenantEntitlementOverride(context.Context, *pb.UpdateTenantEntitlementOverrideRequest) (*pb.TenantSubscriptionResponse, error)
	GetTenantUsage(context.Context, *pb.GetTenantUsageRequest) (*pb.GetTenantUsageResponse, error)
	ListQuotaAlerts(context.Context, *pb.ListQuotaAlertsRequest) (*pb.ListQuotaAlertsResponse, error)
	UpdateQuotaAlert(context.Context, *pb.UpdateQuotaAlertRequest) (*pb.QuotaAlertResponse, error)
}

type Deps struct {
	Auth   AuthAPI
	Admin  AdminAPI
	Audit  AuditAPI
	Tenant TenantAPI
}

type Runtime struct {
	Auth   pb.AuthServiceServer
	Admin  pb.AdminServiceServer
	Tenant pb.PlatformTenantServiceServer
}

func New(deps Deps) (*Runtime, error) {
	if deps.Auth == nil {
		return nil, fmt.Errorf("identity auth api is required")
	}
	if deps.Admin == nil {
		return nil, fmt.Errorf("identity admin api is required")
	}
	if deps.Audit == nil {
		return nil, fmt.Errorf("identity audit api is required")
	}
	if deps.Tenant == nil {
		return nil, fmt.Errorf("identity tenant api is required")
	}
	return &Runtime{
		Auth:   authServer{api: deps.Auth},
		Admin:  adminServer{admin: deps.Admin, audit: deps.Audit},
		Tenant: tenantServer{api: deps.Tenant},
	}, nil
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc service registrar is required")
	}
	if r == nil || r.Auth == nil || r.Admin == nil || r.Tenant == nil {
		return fmt.Errorf("identity runtime is not initialized")
	}
	pb.RegisterAuthServiceServer(registrar, r.Auth)
	pb.RegisterAdminServiceServer(registrar, r.Admin)
	pb.RegisterPlatformTenantServiceServer(registrar, r.Tenant)
	return nil
}

type authServer struct {
	pb.UnimplementedAuthServiceServer
	api AuthAPI
}

func (s authServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return s.api.Register(ctx, req)
}

func (s authServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return s.api.Login(ctx, req)
}

func (s authServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return s.api.RefreshToken(ctx, req)
}

func (s authServer) SwitchTenant(ctx context.Context, req *pb.SwitchTenantRequest) (*pb.LoginResponse, error) {
	return s.api.SwitchTenant(ctx, req)
}

func (s authServer) RevokeRefreshToken(ctx context.Context, req *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error) {
	return s.api.RevokeRefreshToken(ctx, req)
}

func (s authServer) RecordAuthDecision(ctx context.Context, req *pb.AuthAuditRequest) (*pb.CommonResponse, error) {
	return s.api.RecordAuthDecision(ctx, req)
}

func (s authServer) GetPrincipal(ctx context.Context, req *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error) {
	return s.api.GetPrincipal(ctx, req)
}

func (s authServer) AuthorizeInternal(ctx context.Context, req *pb.AuthorizeInternalRequest) (*pb.AuthorizeInternalResponse, error) {
	return s.api.AuthorizeInternal(ctx, req)
}

func (s authServer) UpdateEmail(ctx context.Context, req *pb.UpdateEmailRequest) (*pb.CommonResponse, error) {
	return s.api.UpdateEmail(ctx, req)
}

type adminServer struct {
	pb.UnimplementedAdminServiceServer
	admin AdminAPI
	audit AuditAPI
}

type tenantServer struct {
	pb.UnimplementedPlatformTenantServiceServer
	api TenantAPI
}

func (s tenantServer) CreateTenant(ctx context.Context, req *pb.CreateTenantRequest) (*pb.TenantResponse, error) {
	return s.api.CreateTenant(ctx, req)
}
func (s tenantServer) ListTenants(ctx context.Context, req *pb.ListTenantsRequest) (*pb.ListTenantsResponse, error) {
	return s.api.ListTenants(ctx, req)
}
func (s tenantServer) GetTenant(ctx context.Context, req *pb.GetTenantRequest) (*pb.TenantResponse, error) {
	return s.api.GetTenant(ctx, req)
}
func (s tenantServer) UpdateTenantStatus(ctx context.Context, req *pb.UpdateTenantStatusRequest) (*pb.TenantResponse, error) {
	return s.api.UpdateTenantStatus(ctx, req)
}
func (s tenantServer) ListTenantMemberships(ctx context.Context, req *pb.ListTenantMembershipsRequest) (*pb.ListTenantMembershipsResponse, error) {
	return s.api.ListTenantMemberships(ctx, req)
}
func (s tenantServer) UpdateTenantMembershipStatus(ctx context.Context, req *pb.UpdateTenantMembershipStatusRequest) (*pb.TenantMembershipResponse, error) {
	return s.api.UpdateTenantMembershipStatus(ctx, req)
}
func (s tenantServer) GetPlatformDashboard(ctx context.Context, req *pb.GetPlatformDashboardRequest) (*pb.GetPlatformDashboardResponse, error) {
	return s.api.GetPlatformDashboard(ctx, req)
}
func (s tenantServer) QueryPlatformAuditLogs(ctx context.Context, req *pb.QueryPlatformAuditLogsRequest) (*pb.QueryPlatformAuditLogsResponse, error) {
	return s.api.QueryPlatformAuditLogs(ctx, req)
}
func (s tenantServer) ListPlatformPlans(ctx context.Context, req *pb.ListPlatformPlansRequest) (*pb.ListPlatformPlansResponse, error) {
	return s.api.ListPlatformPlans(ctx, req)
}
func (s tenantServer) SavePlatformPlanVersion(ctx context.Context, req *pb.SavePlatformPlanVersionRequest) (*pb.PlatformPlanVersionResponse, error) {
	return s.api.SavePlatformPlanVersion(ctx, req)
}
func (s tenantServer) PublishPlatformPlanVersion(ctx context.Context, req *pb.PublishPlatformPlanVersionRequest) (*pb.PlatformPlanVersionResponse, error) {
	return s.api.PublishPlatformPlanVersion(ctx, req)
}
func (s tenantServer) GetTenantSubscription(ctx context.Context, req *pb.GetTenantSubscriptionRequest) (*pb.TenantSubscriptionResponse, error) {
	return s.api.GetTenantSubscription(ctx, req)
}
func (s tenantServer) UpdateTenantSubscription(ctx context.Context, req *pb.UpdateTenantSubscriptionRequest) (*pb.TenantSubscriptionResponse, error) {
	return s.api.UpdateTenantSubscription(ctx, req)
}
func (s tenantServer) UpdateTenantEntitlementOverride(ctx context.Context, req *pb.UpdateTenantEntitlementOverrideRequest) (*pb.TenantSubscriptionResponse, error) {
	return s.api.UpdateTenantEntitlementOverride(ctx, req)
}
func (s tenantServer) GetTenantUsage(ctx context.Context, req *pb.GetTenantUsageRequest) (*pb.GetTenantUsageResponse, error) {
	return s.api.GetTenantUsage(ctx, req)
}
func (s tenantServer) ListQuotaAlerts(ctx context.Context, req *pb.ListQuotaAlertsRequest) (*pb.ListQuotaAlertsResponse, error) {
	return s.api.ListQuotaAlerts(ctx, req)
}
func (s tenantServer) UpdateQuotaAlert(ctx context.Context, req *pb.UpdateQuotaAlertRequest) (*pb.QuotaAlertResponse, error) {
	return s.api.UpdateQuotaAlert(ctx, req)
}

func (s adminServer) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	return s.admin.ListRoles(ctx, req)
}

func (s adminServer) ListPermissions(ctx context.Context, req *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	return s.admin.ListPermissions(ctx, req)
}

func (s adminServer) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	return s.admin.GetUserRoles(ctx, req)
}

func (s adminServer) AssignUserRole(ctx context.Context, req *pb.AssignUserRoleRequest) (*pb.CommonResponse, error) {
	return s.admin.AssignUserRole(ctx, req)
}

func (s adminServer) RevokeUserRole(ctx context.Context, req *pb.RevokeUserRoleRequest) (*pb.CommonResponse, error) {
	return s.admin.RevokeUserRole(ctx, req)
}

func (s adminServer) AssignDataScope(ctx context.Context, req *pb.AssignDataScopeRequest) (*pb.CommonResponse, error) {
	return s.admin.AssignDataScope(ctx, req)
}

func (s adminServer) RevokeDataScope(ctx context.Context, req *pb.RevokeDataScopeRequest) (*pb.CommonResponse, error) {
	return s.admin.RevokeDataScope(ctx, req)
}

func (s adminServer) ListStaffUsers(ctx context.Context, req *pb.ListStaffUsersRequest) (*pb.ListStaffUsersResponse, error) {
	return s.admin.ListStaffUsers(ctx, req)
}

func (s adminServer) CreateStaffUser(ctx context.Context, req *pb.CreateStaffUserRequest) (*pb.CreateStaffUserResponse, error) {
	return s.admin.CreateStaffUser(ctx, req)
}
func (s adminServer) ListPlatformUsers(ctx context.Context, req *pb.ListPlatformUsersRequest) (*pb.ListPlatformUsersResponse, error) {
	return s.admin.ListPlatformUsers(ctx, req)
}
func (s adminServer) CreatePlatformUser(ctx context.Context, req *pb.CreatePlatformUserRequest) (*pb.CreatePlatformUserResponse, error) {
	return s.admin.CreatePlatformUser(ctx, req)
}
func (s adminServer) UpdatePlatformUser(ctx context.Context, req *pb.UpdatePlatformUserRequest) (*pb.PlatformUserResponse, error) {
	return s.admin.UpdatePlatformUser(ctx, req)
}

func (s adminServer) QueryAuthAuditLogs(ctx context.Context, req *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error) {
	return s.audit.QueryAuthAuditLogs(ctx, req)
}
