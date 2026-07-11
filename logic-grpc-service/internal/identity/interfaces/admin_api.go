package interfaces

import (
	"context"

	"logic-grpc-service/recruitment/pb"
)

// AdminAPI is the Identity-owned subset of the current AdminService gRPC surface.
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

// AuditAPI is the Identity-owned security-audit query subset currently implemented by AnalyticsService.
type AuditAPI interface {
	QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error)
}
