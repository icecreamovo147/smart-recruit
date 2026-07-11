package runtime

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/recruitment/pb"
)

func TestRuntimeRegistersIdentityGRPCServices(t *testing.T) {
	t.Parallel()

	runtime, err := New(Deps{
		Auth:  &runtimeAuthAPI{},
		Admin: &runtimeAdminAPI{},
		Audit: &runtimeAuditAPI{},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	server := grpc.NewServer()
	t.Cleanup(server.Stop)

	if err := runtime.RegisterGRPC(server); err != nil {
		t.Fatalf("RegisterGRPC() error = %v", err)
	}

	services := server.GetServiceInfo()
	if _, ok := services[pb.AuthService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services missing %s: %v", pb.AuthService_ServiceDesc.ServiceName, services)
	}
	if _, ok := services[pb.AdminService_ServiceDesc.ServiceName]; !ok {
		t.Fatalf("registered services missing %s: %v", pb.AdminService_ServiceDesc.ServiceName, services)
	}
}

func TestRuntimeRequiresIdentityDependencies(t *testing.T) {
	t.Parallel()

	if _, err := New(Deps{}); err == nil {
		t.Fatal("New(Deps{}) error = nil")
	}
}

type runtimeAuthAPI struct{}

func (runtimeAuthAPI) Register(context.Context, *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return &pb.RegisterResponse{Code: errs.OK}, nil
}

func (runtimeAuthAPI) Login(context.Context, *pb.LoginRequest) (*pb.LoginResponse, error) {
	return &pb.LoginResponse{Code: errs.OK}, nil
}

func (runtimeAuthAPI) RefreshToken(context.Context, *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return &pb.RefreshTokenResponse{Code: errs.OK}, nil
}

func (runtimeAuthAPI) RevokeRefreshToken(context.Context, *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeAuthAPI) GetPrincipal(context.Context, *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error) {
	return &pb.GetPrincipalResponse{Code: errs.OK}, nil
}

func (runtimeAuthAPI) UpdateEmail(context.Context, *pb.UpdateEmailRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeAuthAPI) RecordAuthDecision(context.Context, *pb.AuthAuditRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

type runtimeAdminAPI struct{}

func (runtimeAdminAPI) ListRoles(context.Context, *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	return &pb.ListRolesResponse{Code: errs.OK}, nil
}

func (runtimeAdminAPI) ListPermissions(context.Context, *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	return &pb.ListPermissionsResponse{Code: errs.OK}, nil
}

func (runtimeAdminAPI) GetUserRoles(context.Context, *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	return &pb.GetUserRolesResponse{Code: errs.OK}, nil
}

func (runtimeAdminAPI) AssignUserRole(context.Context, *pb.AssignUserRoleRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeAdminAPI) RevokeUserRole(context.Context, *pb.RevokeUserRoleRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeAdminAPI) AssignDataScope(context.Context, *pb.AssignDataScopeRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeAdminAPI) RevokeDataScope(context.Context, *pb.RevokeDataScopeRequest) (*pb.CommonResponse, error) {
	return &pb.CommonResponse{Code: errs.OK}, nil
}

func (runtimeAdminAPI) ListStaffUsers(context.Context, *pb.ListStaffUsersRequest) (*pb.ListStaffUsersResponse, error) {
	return &pb.ListStaffUsersResponse{Code: errs.OK}, nil
}

func (runtimeAdminAPI) CreateStaffUser(context.Context, *pb.CreateStaffUserRequest) (*pb.CreateStaffUserResponse, error) {
	return &pb.CreateStaffUserResponse{Code: errs.OK}, nil
}

type runtimeAuditAPI struct{}

func (runtimeAuditAPI) QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error) {
	return &pb.QueryAuthAuditLogsResponse{Code: errs.OK}, nil
}
