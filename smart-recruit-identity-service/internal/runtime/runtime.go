package runtime

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"logic-grpc-service/recruitment/pb"
)

const ServiceName = "identity-service"

type AuthAPI interface {
	Register(context.Context, *pb.RegisterRequest) (*pb.RegisterResponse, error)
	Login(context.Context, *pb.LoginRequest) (*pb.LoginResponse, error)
	RefreshToken(context.Context, *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error)
	RevokeRefreshToken(context.Context, *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error)
	RecordAuthDecision(context.Context, *pb.AuthAuditRequest) (*pb.CommonResponse, error)
	GetPrincipal(context.Context, *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error)
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
}

type AuditAPI interface {
	QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error)
}

type Deps struct {
	Auth  AuthAPI
	Admin AdminAPI
	Audit AuditAPI
}

type Runtime struct {
	Auth  pb.AuthServiceServer
	Admin pb.AdminServiceServer
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
	return &Runtime{
		Auth:  authServer{api: deps.Auth},
		Admin: adminServer{admin: deps.Admin, audit: deps.Audit},
	}, nil
}

func (r *Runtime) RegisterGRPC(registrar grpc.ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("grpc service registrar is required")
	}
	if r == nil || r.Auth == nil || r.Admin == nil {
		return fmt.Errorf("identity runtime is not initialized")
	}
	pb.RegisterAuthServiceServer(registrar, r.Auth)
	pb.RegisterAdminServiceServer(registrar, r.Admin)
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

func (s authServer) RevokeRefreshToken(ctx context.Context, req *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error) {
	return s.api.RevokeRefreshToken(ctx, req)
}

func (s authServer) RecordAuthDecision(ctx context.Context, req *pb.AuthAuditRequest) (*pb.CommonResponse, error) {
	return s.api.RecordAuthDecision(ctx, req)
}

func (s authServer) GetPrincipal(ctx context.Context, req *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error) {
	return s.api.GetPrincipal(ctx, req)
}

func (s authServer) UpdateEmail(ctx context.Context, req *pb.UpdateEmailRequest) (*pb.CommonResponse, error) {
	return s.api.UpdateEmail(ctx, req)
}

type adminServer struct {
	pb.UnimplementedAdminServiceServer
	admin AdminAPI
	audit AuditAPI
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

func (s adminServer) QueryAuthAuditLogs(ctx context.Context, req *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error) {
	return s.audit.QueryAuthAuditLogs(ctx, req)
}
