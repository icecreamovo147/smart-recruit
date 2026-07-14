package rpc

import (
	"context"

	"google.golang.org/grpc"

	"smart-recruit-proto/recruitment/pb"
)

type recruitmentAdminClient struct {
	pb.AdminServiceClient
	recruitment pb.AdminServiceClient
}

func newRecruitmentAdminClient(base, recruitment pb.AdminServiceClient) pb.AdminServiceClient {
	if recruitment == nil {
		return base
	}
	return &recruitmentAdminClient{
		AdminServiceClient: base,
		recruitment:        recruitment,
	}
}

func (c *recruitmentAdminClient) CreateInviteCode(ctx context.Context, in *pb.CreateInviteCodeRequest, opts ...grpc.CallOption) (*pb.CreateInviteCodeResponse, error) {
	return c.recruitment.CreateInviteCode(ctx, in, opts...)
}

func (c *recruitmentAdminClient) ListInviteCodes(ctx context.Context, in *pb.ListInviteCodesRequest, opts ...grpc.CallOption) (*pb.ListInviteCodesResponse, error) {
	return c.recruitment.ListInviteCodes(ctx, in, opts...)
}

func (c *recruitmentAdminClient) ExtendInviteCode(ctx context.Context, in *pb.ExtendInviteCodeRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	return c.recruitment.ExtendInviteCode(ctx, in, opts...)
}

func (c *recruitmentAdminClient) RevokeInviteCode(ctx context.Context, in *pb.RevokeInviteCodeRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	return c.recruitment.RevokeInviteCode(ctx, in, opts...)
}

func (c *recruitmentAdminClient) ReactivateInviteCode(ctx context.Context, in *pb.ReactivateInviteCodeRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	return c.recruitment.ReactivateInviteCode(ctx, in, opts...)
}

func (c *recruitmentAdminClient) ValidateInviteCode(ctx context.Context, in *pb.ValidateInviteCodeRequest, opts ...grpc.CallOption) (*pb.ValidateInviteCodeResponse, error) {
	return c.recruitment.ValidateInviteCode(ctx, in, opts...)
}

func (c *recruitmentAdminClient) ListDepartments(ctx context.Context, in *pb.ListDepartmentsRequest, opts ...grpc.CallOption) (*pb.ListDepartmentsResponse, error) {
	return c.recruitment.ListDepartments(ctx, in, opts...)
}

func (c *recruitmentAdminClient) CreateDepartment(ctx context.Context, in *pb.CreateDepartmentRequest, opts ...grpc.CallOption) (*pb.DepartmentResponse, error) {
	return c.recruitment.CreateDepartment(ctx, in, opts...)
}

func (c *recruitmentAdminClient) UpdateDepartment(ctx context.Context, in *pb.UpdateDepartmentRequest, opts ...grpc.CallOption) (*pb.DepartmentResponse, error) {
	return c.recruitment.UpdateDepartment(ctx, in, opts...)
}

func (c *recruitmentAdminClient) UpdateDepartmentStatus(ctx context.Context, in *pb.UpdateDepartmentStatusRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	return c.recruitment.UpdateDepartmentStatus(ctx, in, opts...)
}

func (c *recruitmentAdminClient) DeleteDepartment(ctx context.Context, in *pb.DeleteDepartmentRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	return c.recruitment.DeleteDepartment(ctx, in, opts...)
}

func (c *recruitmentAdminClient) ListJobLocations(ctx context.Context, in *pb.ListJobLocationsRequest, opts ...grpc.CallOption) (*pb.ListJobLocationsResponse, error) {
	return c.recruitment.ListJobLocations(ctx, in, opts...)
}

func (c *recruitmentAdminClient) CreateJobLocation(ctx context.Context, in *pb.CreateJobLocationRequest, opts ...grpc.CallOption) (*pb.JobLocationResponse, error) {
	return c.recruitment.CreateJobLocation(ctx, in, opts...)
}

func (c *recruitmentAdminClient) UpdateJobLocation(ctx context.Context, in *pb.UpdateJobLocationRequest, opts ...grpc.CallOption) (*pb.JobLocationResponse, error) {
	return c.recruitment.UpdateJobLocation(ctx, in, opts...)
}

func (c *recruitmentAdminClient) UpdateJobLocationStatus(ctx context.Context, in *pb.UpdateJobLocationStatusRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	return c.recruitment.UpdateJobLocationStatus(ctx, in, opts...)
}

func (c *recruitmentAdminClient) DeleteJobLocation(ctx context.Context, in *pb.DeleteJobLocationRequest, opts ...grpc.CallOption) (*pb.CommonResponse, error) {
	return c.recruitment.DeleteJobLocation(ctx, in, opts...)
}

func (c *recruitmentAdminClient) GetDepartmentLocationConfig(ctx context.Context, in *pb.GetDepartmentLocationConfigRequest, opts ...grpc.CallOption) (*pb.DepartmentLocationConfigResponse, error) {
	return c.recruitment.GetDepartmentLocationConfig(ctx, in, opts...)
}

func (c *recruitmentAdminClient) UpdateDepartmentLocationConfig(ctx context.Context, in *pb.UpdateDepartmentLocationConfigRequest, opts ...grpc.CallOption) (*pb.DepartmentLocationConfigResponse, error) {
	return c.recruitment.UpdateDepartmentLocationConfig(ctx, in, opts...)
}

func (c *recruitmentAdminClient) ListDepartmentsLocationMap(ctx context.Context, in *pb.ListDepartmentsLocationMapRequest, opts ...grpc.CallOption) (*pb.ListDepartmentsLocationMapResponse, error) {
	return c.recruitment.ListDepartmentsLocationMap(ctx, in, opts...)
}

func (c *recruitmentAdminClient) QueryUsageLogs(ctx context.Context, in *pb.QueryUsageLogsRequest, opts ...grpc.CallOption) (*pb.QueryUsageLogsResponse, error) {
	return c.recruitment.QueryUsageLogs(ctx, in, opts...)
}

func (c *recruitmentAdminClient) GetUsageStats(ctx context.Context, in *pb.GetUsageStatsRequest, opts ...grpc.CallOption) (*pb.GetUsageStatsResponse, error) {
	return c.recruitment.GetUsageStats(ctx, in, opts...)
}

func (c *recruitmentAdminClient) GetUsageTrend(ctx context.Context, in *pb.GetUsageTrendRequest, opts ...grpc.CallOption) (*pb.GetUsageTrendResponse, error) {
	return c.recruitment.GetUsageTrend(ctx, in, opts...)
}
