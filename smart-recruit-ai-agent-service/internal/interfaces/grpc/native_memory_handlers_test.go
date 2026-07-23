package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
	platformmetadata "smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

func TestMemoryOwnerFromRequestEnforcesTenantBoundary(t *testing.T) {
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{
		TenantID: 11, UserID: 7, AccountType: "staff",
	})
	owner, err := memoryOwnerFromRequest(ctx, 11, int32(domainmemory.OwnerRoleHR), 7)
	if err != nil || owner.TenantID == nil || *owner.TenantID != 11 {
		t.Fatalf("owner = %+v err=%v", owner, err)
	}
	if _, err := memoryOwnerFromRequest(ctx, 12, int32(domainmemory.OwnerRoleHR), 7); status.Code(err) != codes.NotFound {
		t.Fatalf("mismatched tenant error = %v", err)
	}
	if _, err := memoryOwnerFromRequest(ctx, 0, int32(domainmemory.OwnerRoleHR), 7); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing tenant error = %v", err)
	}
	if _, err := memoryOwnerFromRequest(ctx, 11, int32(domainmemory.OwnerRoleCandidate), 7); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("candidate tenant error = %v", err)
	}
	candidate, err := memoryOwnerFromRequest(context.Background(), 0, int32(domainmemory.OwnerRoleCandidate), 7)
	if err != nil || candidate.TenantID != nil {
		t.Fatalf("candidate owner = %+v err=%v", candidate, err)
	}
}

func TestDebugSemanticRetrievalRejectsForgedMemoryTenant(t *testing.T) {
	ctx := platformmetadata.WithTenantActor(context.Background(), platformmetadata.TenantContext{
		TenantID: 11, UserID: 7, AccountType: "staff",
	})
	_, err := (nativeAgentSkillService{}).DebugSemanticRetrieval(ctx, &pb.DebugSemanticRetrievalRequest{
		TenantId:  12,
		OwnerRole: int32(domainmemory.OwnerRoleHR),
		OwnerId:   7,
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("forged tenant error = %v", err)
	}
}
