package rpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"smart-recruit-proto/recruitment/pb"
)

func TestIdentityAdminClientRoutesRBACAndAuditToIdentity(t *testing.T) {
	base := &recordingAdminClient{name: "base"}
	identity := &recordingAdminClient{name: "identity"}
	client := newIdentityAdminClient(base, identity)

	methods := []struct {
		name string
		call func(context.Context, pb.AdminServiceClient) error
	}{
		{
			name: "ListRoles",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.ListRoles(ctx, &pb.ListRolesRequest{})
				return err
			},
		},
		{
			name: "GetUserRoles",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.GetUserRoles(ctx, &pb.GetUserRolesRequest{})
				return err
			},
		},
		{
			name: "ListPlatformUsers",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.ListPlatformUsers(ctx, &pb.ListPlatformUsersRequest{})
				return err
			},
		},
		{
			name: "CreatePlatformUser",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.CreatePlatformUser(ctx, &pb.CreatePlatformUserRequest{})
				return err
			},
		},
		{
			name: "UpdatePlatformUser",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.UpdatePlatformUser(ctx, &pb.UpdatePlatformUserRequest{})
				return err
			},
		},
		{
			name: "QueryAuthAuditLogs",
			call: func(ctx context.Context, c pb.AdminServiceClient) error {
				_, err := c.QueryAuthAuditLogs(ctx, &pb.QueryAuthAuditLogsRequest{})
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
		t.Fatalf("base admin client received identity-owned calls: %v", base.calls)
	}
	for _, method := range methods {
		if !identity.called(method.name) {
			t.Fatalf("identity admin client did not receive %s; calls=%v", method.name, identity.calls)
		}
	}
}

type recordingAdminClient struct {
	pb.AdminServiceClient
	name  string
	calls []string
}

func (c *recordingAdminClient) called(method string) bool {
	for _, call := range c.calls {
		if call == method {
			return true
		}
	}
	return false
}

func (c *recordingAdminClient) ListRoles(context.Context, *pb.ListRolesRequest, ...grpc.CallOption) (*pb.ListRolesResponse, error) {
	c.calls = append(c.calls, "ListRoles")
	return &pb.ListRolesResponse{}, nil
}

func (c *recordingAdminClient) GetUserRoles(context.Context, *pb.GetUserRolesRequest, ...grpc.CallOption) (*pb.GetUserRolesResponse, error) {
	c.calls = append(c.calls, "GetUserRoles")
	return &pb.GetUserRolesResponse{}, nil
}

func (c *recordingAdminClient) ListPlatformUsers(context.Context, *pb.ListPlatformUsersRequest, ...grpc.CallOption) (*pb.ListPlatformUsersResponse, error) {
	c.calls = append(c.calls, "ListPlatformUsers")
	return &pb.ListPlatformUsersResponse{}, nil
}

func (c *recordingAdminClient) CreatePlatformUser(context.Context, *pb.CreatePlatformUserRequest, ...grpc.CallOption) (*pb.CreatePlatformUserResponse, error) {
	c.calls = append(c.calls, "CreatePlatformUser")
	return &pb.CreatePlatformUserResponse{}, nil
}

func (c *recordingAdminClient) UpdatePlatformUser(context.Context, *pb.UpdatePlatformUserRequest, ...grpc.CallOption) (*pb.PlatformUserResponse, error) {
	c.calls = append(c.calls, "UpdatePlatformUser")
	return &pb.PlatformUserResponse{}, nil
}

func (c *recordingAdminClient) QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest, ...grpc.CallOption) (*pb.QueryAuthAuditLogsResponse, error) {
	c.calls = append(c.calls, "QueryAuthAuditLogs")
	return &pb.QueryAuthAuditLogsResponse{}, nil
}
