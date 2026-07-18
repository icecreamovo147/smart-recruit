package rpc

import (
	"context"

	"google.golang.org/grpc"

	"smart-recruit-proto/recruitment/pb"
)

type identityAdminClient struct {
	pb.AdminServiceClient
	identity pb.AdminServiceClient
}

func newIdentityAdminClient(base, identity pb.AdminServiceClient) pb.AdminServiceClient {
	if identity == nil {
		return base
	}
	return &identityAdminClient{
		AdminServiceClient: base,
		identity:           identity,
	}
}

func (c *identityAdminClient) ListRoles(ctx context.Context, in *pb.ListRolesRequest, opts ...grpc.CallOption) (*pb.ListRolesResponse, error) {
	return c.identity.ListRoles(ctx, in, opts...)
}

func (c *identityAdminClient) ListPermissions(ctx context.Context, in *pb.ListPermissionsRequest, opts ...grpc.CallOption) (*pb.ListPermissionsResponse, error) {
	return c.identity.ListPermissions(ctx, in, opts...)
}

func (c *identityAdminClient) GetUserRoles(ctx context.Context, in *pb.GetUserRolesRequest, opts ...grpc.CallOption) (*pb.GetUserRolesResponse, error) {
	return c.identity.GetUserRoles(ctx, in, opts...)
}

func (c *identityAdminClient) AssignUserRole(ctx context.Context, in *pb.AssignUserRoleRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	return c.identity.AssignUserRole(ctx, in, opts...)
}

func (c *identityAdminClient) RevokeUserRole(ctx context.Context, in *pb.RevokeUserRoleRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	return c.identity.RevokeUserRole(ctx, in, opts...)
}

func (c *identityAdminClient) AssignDataScope(ctx context.Context, in *pb.AssignDataScopeRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	return c.identity.AssignDataScope(ctx, in, opts...)
}

func (c *identityAdminClient) RevokeDataScope(ctx context.Context, in *pb.RevokeDataScopeRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	return c.identity.RevokeDataScope(ctx, in, opts...)
}

func (c *identityAdminClient) ListStaffUsers(ctx context.Context, in *pb.ListStaffUsersRequest, opts ...grpc.CallOption) (*pb.ListStaffUsersResponse, error) {
	return c.identity.ListStaffUsers(ctx, in, opts...)
}

func (c *identityAdminClient) CreateStaffUser(ctx context.Context, in *pb.CreateStaffUserRequest, opts ...grpc.CallOption) (*pb.CreateStaffUserResponse, error) {
	return c.identity.CreateStaffUser(ctx, in, opts...)
}

func (c *identityAdminClient) QueryAuthAuditLogs(ctx context.Context, in *pb.QueryAuthAuditLogsRequest, opts ...grpc.CallOption) (*pb.QueryAuthAuditLogsResponse, error) {
	return c.identity.QueryAuthAuditLogs(ctx, in, opts...)
}
