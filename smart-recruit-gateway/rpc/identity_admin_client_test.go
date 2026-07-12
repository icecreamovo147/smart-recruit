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

	if _, err := client.ListRoles(context.Background(), &pb.ListRolesRequest{}); err != nil {
		t.Fatalf("ListRoles returned error: %v", err)
	}
	if _, err := client.GetUserRoles(context.Background(), &pb.GetUserRolesRequest{}); err != nil {
		t.Fatalf("GetUserRoles returned error: %v", err)
	}
	if _, err := client.QueryAuthAuditLogs(context.Background(), &pb.QueryAuthAuditLogsRequest{}); err != nil {
		t.Fatalf("QueryAuthAuditLogs returned error: %v", err)
	}
	if len(base.calls) != 0 {
		t.Fatalf("base admin client received identity-owned calls: %v", base.calls)
	}
	for _, method := range []string{"ListRoles", "GetUserRoles", "QueryAuthAuditLogs"} {
		if !identity.called(method) {
			t.Fatalf("identity admin client did not receive %s; calls=%v", method, identity.calls)
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

func (c *recordingAdminClient) QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest, ...grpc.CallOption) (*pb.QueryAuthAuditLogsResponse, error) {
	c.calls = append(c.calls, "QueryAuthAuditLogs")
	return &pb.QueryAuthAuditLogsResponse{}, nil
}
