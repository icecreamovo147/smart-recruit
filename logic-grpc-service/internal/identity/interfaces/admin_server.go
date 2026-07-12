package interfaces

import (
	"context"
	"fmt"

	"logic-grpc-service/recruitment/pb"
)

type AdminServer struct {
	pb.UnimplementedAdminServiceServer
	admin AdminAPI
	audit AuditAPI
}

func NewAdminServer(admin AdminAPI, audit AuditAPI) (*AdminServer, error) {
	if admin == nil {
		return nil, fmt.Errorf("identity admin api is required")
	}
	if audit == nil {
		return nil, fmt.Errorf("identity audit api is required")
	}
	return &AdminServer{admin: admin, audit: audit}, nil
}

func (s *AdminServer) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	return s.admin.ListRoles(ctx, req)
}

func (s *AdminServer) ListPermissions(ctx context.Context, req *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	return s.admin.ListPermissions(ctx, req)
}

func (s *AdminServer) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	return s.admin.GetUserRoles(ctx, req)
}

func (s *AdminServer) AssignUserRole(ctx context.Context, req *pb.AssignUserRoleRequest) (*pb.CommonResponse, error) {
	return s.admin.AssignUserRole(ctx, req)
}

func (s *AdminServer) RevokeUserRole(ctx context.Context, req *pb.RevokeUserRoleRequest) (*pb.CommonResponse, error) {
	return s.admin.RevokeUserRole(ctx, req)
}

func (s *AdminServer) AssignDataScope(ctx context.Context, req *pb.AssignDataScopeRequest) (*pb.CommonResponse, error) {
	return s.admin.AssignDataScope(ctx, req)
}

func (s *AdminServer) RevokeDataScope(ctx context.Context, req *pb.RevokeDataScopeRequest) (*pb.CommonResponse, error) {
	return s.admin.RevokeDataScope(ctx, req)
}

func (s *AdminServer) ListStaffUsers(ctx context.Context, req *pb.ListStaffUsersRequest) (*pb.ListStaffUsersResponse, error) {
	return s.admin.ListStaffUsers(ctx, req)
}

func (s *AdminServer) CreateStaffUser(ctx context.Context, req *pb.CreateStaffUserRequest) (*pb.CreateStaffUserResponse, error) {
	return s.admin.CreateStaffUser(ctx, req)
}

func (s *AdminServer) QueryAuthAuditLogs(ctx context.Context, req *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error) {
	return s.audit.QueryAuthAuditLogs(ctx, req)
}
