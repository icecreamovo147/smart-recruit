package client

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	grpcmetadata "google.golang.org/grpc/metadata"

	"smart-recruit-interview-service/internal/application/port"
	"smart-recruit-interview-service/internal/domain/model"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestApplicationLifecycleAdapterUsesServiceAuthorizationAndPreservesBusinessActor(t *testing.T) {
	client := &capturingApplicationOwnerClient{}
	adapter := NewApplicationLifecycleAdapter(client)
	ctx := grpcmetadata.NewIncomingContext(context.Background(), grpcmetadata.Pairs(
		"x-authenticated-user-id", "3",
		"x-authenticated-account-type", "staff",
		"x-authenticated-tenant-id", "7",
		"x-authenticated-membership-id", "11",
		"x-authenticated-client-app", "staff",
	))

	changed, err := adapter.ApplyTransition(ctx, port.LifecycleTransitionCommand{
		ApplicationID:    6,
		FromStatus:       model.ApplicationStatus("interview_scheduled"),
		ToStatus:         model.ApplicationStatus("interviewing"),
		ActorUserID:      3,
		ActorAccountType: "staff",
		Reason:           "面试官提交反馈",
	})
	if err != nil {
		t.Fatalf("ApplyTransition() error = %v", err)
	}
	if !changed {
		t.Fatal("ApplyTransition() changed = false, want true")
	}

	md, ok := grpcmetadata.FromIncomingContext(client.ctx)
	if !ok {
		t.Fatal("downstream context has no incoming metadata")
	}
	if got := firstMetadataValue(md, "x-authenticated-account-type"); got != "service" {
		t.Fatalf("downstream account type = %q, want service", got)
	}
	if got := firstMetadataValue(md, "x-authenticated-client-app"); got != "interview-service" {
		t.Fatalf("downstream client app = %q, want interview-service", got)
	}
	if got := firstMetadataValue(md, "x-authenticated-tenant-id"); got != "7" {
		t.Fatalf("downstream tenant id = %q, want 7", got)
	}
	if values := md.Get("x-authenticated-user-id"); len(values) != 0 {
		t.Fatalf("downstream user id metadata = %v, want empty", values)
	}
	if values := md.Get("x-authenticated-membership-id"); len(values) != 0 {
		t.Fatalf("downstream membership metadata = %v, want empty", values)
	}
	if client.req.GetActorUserId() != 3 || client.req.GetActorAccountType() != "staff" {
		t.Fatalf("business actor = %d/%s, want 3/staff", client.req.GetActorUserId(), client.req.GetActorAccountType())
	}
}

type capturingApplicationOwnerClient struct {
	ctx context.Context
	req *pb.ApplyApplicationLifecycleTransitionRequest
}

func (c *capturingApplicationOwnerClient) GetApplicationSnapshot(context.Context, *pb.GetApplicationSnapshotRequest, ...grpc.CallOption) (*pb.GetApplicationSnapshotResponse, error) {
	return &pb.GetApplicationSnapshotResponse{Code: errs.OK}, nil
}

func (c *capturingApplicationOwnerClient) ApplyApplicationLifecycleTransition(ctx context.Context, req *pb.ApplyApplicationLifecycleTransitionRequest, _ ...grpc.CallOption) (*pb.ApplyApplicationLifecycleTransitionResponse, error) {
	c.ctx = ctx
	c.req = req
	return &pb.ApplyApplicationLifecycleTransitionResponse{Code: errs.OK, Changed: true}, nil
}

func firstMetadataValue(md grpcmetadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
