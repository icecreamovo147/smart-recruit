package rpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"smart-recruit-proto/recruitment/pb"
)

func TestRecruitmentAdminClientRoutesRecruitmentOwnedMethods(t *testing.T) {
	base := &recordingAdminClient{name: "base"}
	recruitment := &recordingAdminClient{name: "recruitment"}
	client := newRecruitmentAdminClient(base, recruitment)

	methods := []struct {
		name string
		call func(context.Context, pb.AdminServiceClient) error
	}{
		{
			name: "CreateInviteCode",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.CreateInviteCode(ctx, &pb.CreateInviteCodeRequest{})
				return err
			},
		},
		{
			name: "ListInviteCodes",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.ListInviteCodes(ctx, &pb.ListInviteCodesRequest{})
				return err
			},
		},
		{
			name: "ExtendInviteCode",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.ExtendInviteCode(ctx, &pb.ExtendInviteCodeRequest{})
				return err
			},
		},
		{
			name: "RevokeInviteCode",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.RevokeInviteCode(ctx, &pb.RevokeInviteCodeRequest{})
				return err
			},
		},
		{
			name: "ReactivateInviteCode",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.ReactivateInviteCode(ctx, &pb.ReactivateInviteCodeRequest{})
				return err
			},
		},
		{
			name: "ValidateInviteCode",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.ValidateInviteCode(ctx, &pb.ValidateInviteCodeRequest{})
				return err
			},
		},
		{
			name: "ListDepartments",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.ListDepartments(ctx, &pb.ListDepartmentsRequest{})
				return err
			},
		},
		{
			name: "CreateDepartment",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.CreateDepartment(ctx, &pb.CreateDepartmentRequest{})
				return err
			},
		},
		{
			name: "UpdateDepartment",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.UpdateDepartment(ctx, &pb.UpdateDepartmentRequest{})
				return err
			},
		},
		{
			name: "UpdateDepartmentStatus",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.UpdateDepartmentStatus(ctx, &pb.UpdateDepartmentStatusRequest{})
				return err
			},
		},
		{
			name: "DeleteDepartment",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.DeleteDepartment(ctx, &pb.DeleteDepartmentRequest{})
				return err
			},
		},
		{
			name: "ListJobLocations",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.ListJobLocations(ctx, &pb.ListJobLocationsRequest{})
				return err
			},
		},
		{
			name: "CreateJobLocation",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.CreateJobLocation(ctx, &pb.CreateJobLocationRequest{})
				return err
			},
		},
		{
			name: "UpdateJobLocation",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.UpdateJobLocation(ctx, &pb.UpdateJobLocationRequest{})
				return err
			},
		},
		{
			name: "UpdateJobLocationStatus",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.UpdateJobLocationStatus(ctx, &pb.UpdateJobLocationStatusRequest{})
				return err
			},
		},
		{
			name: "DeleteJobLocation",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.DeleteJobLocation(ctx, &pb.DeleteJobLocationRequest{})
				return err
			},
		},
		{
			name: "GetDepartmentLocationConfig",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.GetDepartmentLocationConfig(ctx, &pb.GetDepartmentLocationConfigRequest{})
				return err
			},
		},
		{
			name: "UpdateDepartmentLocationConfig",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.UpdateDepartmentLocationConfig(ctx, &pb.UpdateDepartmentLocationConfigRequest{})
				return err
			},
		},
		{
			name: "ListDepartmentsLocationMap",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.ListDepartmentsLocationMap(ctx, &pb.ListDepartmentsLocationMapRequest{})
				return err
			},
		},
		{
			name: "QueryUsageLogs",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.QueryUsageLogs(ctx, &pb.QueryUsageLogsRequest{})
				return err
			},
		},
		{
			name: "GetUsageStats",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.GetUsageStats(ctx, &pb.GetUsageStatsRequest{})
				return err
			},
		},
		{
			name: "GetUsageTrend",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.GetUsageTrend(ctx, &pb.GetUsageTrendRequest{})
				return err
			},
		},
	}

	for _, method := range methods {
		if err := method.call(context.Background(), client); err != nil {
			t.Fatalf("%s returned error: %v", method.name, err)
		}
	}

	if len(base.calls) != 0 {
		t.Fatalf("base admin client received recruitment-owned calls: %v", base.calls)
	}
	if got, want := len(recruitment.calls), len(methods); got != want {
		t.Fatalf("recruitment admin client received %d calls, want %d; calls=%v", got, want, recruitment.calls)
	}
	for _, method := range methods {
		if !recruitment.called(method.name) {
			t.Fatalf("recruitment admin client did not receive %s; calls=%v", method.name, recruitment.calls)
		}
	}
}

func TestCompositeAdminClientRoutesServiceOwnedMethods(t *testing.T) {
	base := &recordingAdminClient{name: "base"}
	recruitment := &recordingAdminClient{name: "recruitment"}
	identity := &recordingAdminClient{name: "identity"}
	analytics := &recordingAdminClient{name: "analytics"}
	client := newAnalyticsAdminClient(
		newIdentityAdminClient(
			newRecruitmentAdminClient(base, recruitment),
			identity,
		),
		analytics,
	)

	if _, err := client.CreateInviteCode(context.Background(), &pb.CreateInviteCodeRequest{}); err != nil {
		t.Fatalf("CreateInviteCode returned error: %v", err)
	}
	if _, err := client.QueryUsageLogs(context.Background(), &pb.QueryUsageLogsRequest{}); err != nil {
		t.Fatalf("QueryUsageLogs returned error: %v", err)
	}
	if _, err := client.GetUsageStats(context.Background(), &pb.GetUsageStatsRequest{}); err != nil {
		t.Fatalf("GetUsageStats returned error: %v", err)
	}
	if _, err := client.ListRoles(context.Background(), &pb.ListRolesRequest{}); err != nil {
		t.Fatalf("ListRoles returned error: %v", err)
	}
	if _, err := client.QueryAuthAuditLogs(context.Background(), &pb.QueryAuthAuditLogsRequest{}); err != nil {
		t.Fatalf("QueryAuthAuditLogs returned error: %v", err)
	}
	if _, err := client.GetDashboardReport(context.Background(), &pb.GetDashboardReportRequest{}); err != nil {
		t.Fatalf("GetDashboardReport returned error: %v", err)
	}

	if len(base.calls) != 0 {
		t.Fatalf("base admin client received service-owned calls: %v", base.calls)
	}
	for _, method := range []string{"CreateInviteCode", "QueryUsageLogs", "GetUsageStats"} {
		if !recruitment.called(method) {
			t.Fatalf("recruitment admin client did not receive %s; calls=%v", method, recruitment.calls)
		}
	}
	for _, method := range []string{"ListRoles", "QueryAuthAuditLogs"} {
		if !identity.called(method) {
			t.Fatalf("identity admin client did not receive %s; calls=%v", method, identity.calls)
		}
	}
	if !analytics.called("GetDashboardReport") {
		t.Fatalf("analytics admin client did not receive GetDashboardReport; calls=%v", analytics.calls)
	}
	if recruitment.called("ListRoles") || recruitment.called("QueryAuthAuditLogs") || recruitment.called("GetDashboardReport") {
		t.Fatalf("recruitment admin client received non-recruitment calls: %v", recruitment.calls)
	}
	if identity.called("CreateInviteCode") || identity.called("QueryUsageLogs") || identity.called("GetDashboardReport") {
		t.Fatalf("identity admin client received non-identity calls: %v", identity.calls)
	}
	if analytics.called("CreateInviteCode") || analytics.called("QueryUsageLogs") || analytics.called("ListRoles") {
		t.Fatalf("analytics admin client received non-analytics calls: %v", analytics.calls)
	}
}

func (c *recordingAdminClient) CreateInviteCode(context.Context, *pb.CreateInviteCodeRequest, ...grpc.CallOption) (*pb.CreateInviteCodeResponse, error) {
	c.calls = append(c.calls, "CreateInviteCode")
	return &pb.CreateInviteCodeResponse{}, nil
}

func (c *recordingAdminClient) ListInviteCodes(context.Context, *pb.ListInviteCodesRequest, ...grpc.CallOption) (*pb.ListInviteCodesResponse, error) {
	c.calls = append(c.calls, "ListInviteCodes")
	return &pb.ListInviteCodesResponse{}, nil
}

func (c *recordingAdminClient) ExtendInviteCode(context.Context, *pb.ExtendInviteCodeRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	c.calls = append(c.calls, "ExtendInviteCode")
	return &pb.CommonResponse{}, nil
}

func (c *recordingAdminClient) RevokeInviteCode(context.Context, *pb.RevokeInviteCodeRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	c.calls = append(c.calls, "RevokeInviteCode")
	return &pb.CommonResponse{}, nil
}

func (c *recordingAdminClient) ReactivateInviteCode(context.Context, *pb.ReactivateInviteCodeRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	c.calls = append(c.calls, "ReactivateInviteCode")
	return &pb.CommonResponse{}, nil
}

func (c *recordingAdminClient) ValidateInviteCode(context.Context, *pb.ValidateInviteCodeRequest, ...grpc.CallOption) (*pb.ValidateInviteCodeResponse, error) {
	c.calls = append(c.calls, "ValidateInviteCode")
	return &pb.ValidateInviteCodeResponse{}, nil
}

func (c *recordingAdminClient) ListDepartments(context.Context, *pb.ListDepartmentsRequest, ...grpc.CallOption) (*pb.ListDepartmentsResponse, error) {
	c.calls = append(c.calls, "ListDepartments")
	return &pb.ListDepartmentsResponse{}, nil
}

func (c *recordingAdminClient) CreateDepartment(context.Context, *pb.CreateDepartmentRequest, ...grpc.CallOption) (*pb.DepartmentResponse, error) {
	c.calls = append(c.calls, "CreateDepartment")
	return &pb.DepartmentResponse{}, nil
}

func (c *recordingAdminClient) UpdateDepartment(context.Context, *pb.UpdateDepartmentRequest, ...grpc.CallOption) (*pb.DepartmentResponse, error) {
	c.calls = append(c.calls, "UpdateDepartment")
	return &pb.DepartmentResponse{}, nil
}

func (c *recordingAdminClient) UpdateDepartmentStatus(context.Context, *pb.UpdateDepartmentStatusRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	c.calls = append(c.calls, "UpdateDepartmentStatus")
	return &pb.CommonResponse{}, nil
}

func (c *recordingAdminClient) DeleteDepartment(context.Context, *pb.DeleteDepartmentRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	c.calls = append(c.calls, "DeleteDepartment")
	return &pb.CommonResponse{}, nil
}

func (c *recordingAdminClient) ListJobLocations(context.Context, *pb.ListJobLocationsRequest, ...grpc.CallOption) (*pb.ListJobLocationsResponse, error) {
	c.calls = append(c.calls, "ListJobLocations")
	return &pb.ListJobLocationsResponse{}, nil
}

func (c *recordingAdminClient) CreateJobLocation(context.Context, *pb.CreateJobLocationRequest, ...grpc.CallOption) (*pb.JobLocationResponse, error) {
	c.calls = append(c.calls, "CreateJobLocation")
	return &pb.JobLocationResponse{}, nil
}

func (c *recordingAdminClient) UpdateJobLocation(context.Context, *pb.UpdateJobLocationRequest, ...grpc.CallOption) (*pb.JobLocationResponse, error) {
	c.calls = append(c.calls, "UpdateJobLocation")
	return &pb.JobLocationResponse{}, nil
}

func (c *recordingAdminClient) UpdateJobLocationStatus(context.Context, *pb.UpdateJobLocationStatusRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	c.calls = append(c.calls, "UpdateJobLocationStatus")
	return &pb.CommonResponse{}, nil
}

func (c *recordingAdminClient) DeleteJobLocation(context.Context, *pb.DeleteJobLocationRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	c.calls = append(c.calls, "DeleteJobLocation")
	return &pb.CommonResponse{}, nil
}

func (c *recordingAdminClient) GetDepartmentLocationConfig(context.Context, *pb.GetDepartmentLocationConfigRequest, ...grpc.CallOption) (*pb.DepartmentLocationConfigResponse, error) {
	c.calls = append(c.calls, "GetDepartmentLocationConfig")
	return &pb.DepartmentLocationConfigResponse{}, nil
}

func (c *recordingAdminClient) UpdateDepartmentLocationConfig(context.Context, *pb.UpdateDepartmentLocationConfigRequest, ...grpc.CallOption) (*pb.DepartmentLocationConfigResponse, error) {
	c.calls = append(c.calls, "UpdateDepartmentLocationConfig")
	return &pb.DepartmentLocationConfigResponse{}, nil
}

func (c *recordingAdminClient) ListDepartmentsLocationMap(context.Context, *pb.ListDepartmentsLocationMapRequest, ...grpc.CallOption) (*pb.ListDepartmentsLocationMapResponse, error) {
	c.calls = append(c.calls, "ListDepartmentsLocationMap")
	return &pb.ListDepartmentsLocationMapResponse{}, nil
}

func (c *recordingAdminClient) QueryUsageLogs(context.Context, *pb.QueryUsageLogsRequest, ...grpc.CallOption) (*pb.QueryUsageLogsResponse, error) {
	c.calls = append(c.calls, "QueryUsageLogs")
	return &pb.QueryUsageLogsResponse{}, nil
}

func (c *recordingAdminClient) GetUsageStats(context.Context, *pb.GetUsageStatsRequest, ...grpc.CallOption) (*pb.GetUsageStatsResponse, error) {
	c.calls = append(c.calls, "GetUsageStats")
	return &pb.GetUsageStatsResponse{}, nil
}

func (c *recordingAdminClient) GetUsageTrend(context.Context, *pb.GetUsageTrendRequest, ...grpc.CallOption) (*pb.GetUsageTrendResponse, error) {
	c.calls = append(c.calls, "GetUsageTrend")
	return &pb.GetUsageTrendResponse{}, nil
}
