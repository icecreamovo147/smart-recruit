package interfaces

import (
	"context"
	"testing"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/service"
)

func TestCurrentServicesSatisfyIdentityAPI(t *testing.T) {
	t.Parallel()

	var _ AuthAPI = (*service.AuthService)(nil)
	var _ AdminAPI = (*service.AdminService)(nil)
	var _ AuditAPI = (*service.AnalyticsService)(nil)
}

func TestAuthServerForwardsIdentityAuthAPI(t *testing.T) {
	t.Parallel()

	auth := &fakeAuthAPI{}
	server, err := NewAuthServer(auth)
	if err != nil {
		t.Fatalf("NewAuthServer() error = %v", err)
	}
	if _, err := server.Login(context.Background(), &pb.LoginRequest{}); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if auth.loginCalls != 1 {
		t.Fatalf("login calls = %d, want 1", auth.loginCalls)
	}
}

func TestAdminServerForwardsIdentityAdminAndAuditAPIs(t *testing.T) {
	t.Parallel()

	admin := &fakeAdminAPI{}
	audit := &fakeAuditAPI{}
	server, err := NewAdminServer(admin, audit)
	if err != nil {
		t.Fatalf("NewAdminServer() error = %v", err)
	}
	if _, err := server.AssignUserRole(context.Background(), &pb.AssignUserRoleRequest{}); err != nil {
		t.Fatalf("AssignUserRole() error = %v", err)
	}
	if _, err := server.QueryAuthAuditLogs(context.Background(), &pb.QueryAuthAuditLogsRequest{}); err != nil {
		t.Fatalf("QueryAuthAuditLogs() error = %v", err)
	}
	if admin.assignRoleCalls != 1 {
		t.Fatalf("assign role calls = %d, want 1", admin.assignRoleCalls)
	}
	if audit.queryCalls != 1 {
		t.Fatalf("audit query calls = %d, want 1", audit.queryCalls)
	}
}

func TestIdentityServersRequireDependencies(t *testing.T) {
	t.Parallel()

	if _, err := NewAuthServer(nil); err == nil {
		t.Fatal("NewAuthServer(nil) error = nil")
	}
	if _, err := NewAdminServer(nil, &fakeAuditAPI{}); err == nil {
		t.Fatal("NewAdminServer(nil, audit) error = nil")
	}
	if _, err := NewAdminServer(&fakeAdminAPI{}, nil); err == nil {
		t.Fatal("NewAdminServer(admin, nil) error = nil")
	}
}

type fakeAuthAPI struct {
	loginCalls int
}

func (f *fakeAuthAPI) Register(context.Context, *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return &pb.RegisterResponse{Code: errs.OK}, nil
}

func (f *fakeAuthAPI) Login(context.Context, *pb.LoginRequest) (*pb.LoginResponse, error) {
	f.loginCalls++
	return &pb.LoginResponse{Code: errs.OK}, nil
}

func (f *fakeAuthAPI) RefreshToken(context.Context, *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return &pb.RefreshTokenResponse{Code: errs.OK}, nil
}

func (f *fakeAuthAPI) RevokeRefreshToken(context.Context, *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (f *fakeAuthAPI) GetPrincipal(context.Context, *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error) {
	return &pb.GetPrincipalResponse{Code: errs.OK}, nil
}

func (f *fakeAuthAPI) UpdateEmail(context.Context, *pb.UpdateEmailRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (f *fakeAuthAPI) RecordAuthDecision(context.Context, *pb.AuthAuditRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

type fakeAdminAPI struct {
	assignRoleCalls int
}

func (f *fakeAdminAPI) ListRoles(context.Context, *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	return &pb.ListRolesResponse{Code: errs.OK}, nil
}

func (f *fakeAdminAPI) ListPermissions(context.Context, *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	return &pb.ListPermissionsResponse{Code: errs.OK}, nil
}

func (f *fakeAdminAPI) GetUserRoles(context.Context, *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	return &pb.GetUserRolesResponse{Code: errs.OK}, nil
}

func (f *fakeAdminAPI) AssignUserRole(context.Context, *pb.AssignUserRoleRequest) (*pb.CommonResponse, error) {
	f.assignRoleCalls++
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (f *fakeAdminAPI) RevokeUserRole(context.Context, *pb.RevokeUserRoleRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (f *fakeAdminAPI) AssignDataScope(context.Context, *pb.AssignDataScopeRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (f *fakeAdminAPI) RevokeDataScope(context.Context, *pb.RevokeDataScopeRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (f *fakeAdminAPI) ListStaffUsers(context.Context, *pb.ListStaffUsersRequest) (*pb.ListStaffUsersResponse, error) {
	return &pb.ListStaffUsersResponse{Code: errs.OK}, nil
}

func (f *fakeAdminAPI) CreateStaffUser(context.Context, *pb.CreateStaffUserRequest) (*pb.CreateStaffUserResponse, error) {
	return &pb.CreateStaffUserResponse{Code: errs.OK}, nil
}

type fakeAuditAPI struct {
	queryCalls int
}

func (f *fakeAuditAPI) QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error) {
	f.queryCalls++
	return &pb.QueryAuthAuditLogsResponse{Code: errs.OK}, nil
}
