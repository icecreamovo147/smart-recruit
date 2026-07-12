package runtime

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestRuntimeRegistersIdentityGRPCServices(t *testing.T) {
	runtime, err := New(Deps{
		Auth:  fakeAuthAPI{},
		Admin: fakeAdminAPI{},
		Audit: fakeAuditAPI{},
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

type fakeAuditAPI struct{}

func (fakeAuditAPI) QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error) {
	return &pb.QueryAuthAuditLogsResponse{Code: errs.OK}, nil
}
