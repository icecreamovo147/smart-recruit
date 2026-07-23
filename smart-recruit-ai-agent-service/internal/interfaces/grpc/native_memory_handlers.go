package grpc

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	appmemory "smart-recruit-ai-agent-service/internal/application/memory"
	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
	platformmetadata "smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

func memoryOwnerFromRequest(ctx context.Context, tenantID int64, ownerRole int32, ownerID uint64) (domainmemory.OwnerKey, error) {
	owner := domainmemory.OwnerKey{Role: domainmemory.OwnerRole(ownerRole), ID: ownerID}
	switch owner.Role {
	case domainmemory.OwnerRoleHR:
		if tenantID <= 0 {
			return domainmemory.OwnerKey{}, status.Error(codes.InvalidArgument, "tenant_id is required for HR memory")
		}
		trustedTenantID := platformmetadata.GetTenantContext(ctx).TenantID
		if trustedTenantID > 0 && trustedTenantID != tenantID {
			return domainmemory.OwnerKey{}, status.Error(codes.NotFound, "memory not found")
		}
		value := uint64(tenantID)
		owner.TenantID = &value
	case domainmemory.OwnerRoleCandidate:
		if tenantID != 0 {
			return domainmemory.OwnerKey{}, status.Error(codes.InvalidArgument, "candidate memory must not include tenant_id")
		}
	default:
		return domainmemory.OwnerKey{}, status.Error(codes.InvalidArgument, "invalid owner_role")
	}
	if err := owner.Validate(); err != nil {
		return domainmemory.OwnerKey{}, status.Error(codes.InvalidArgument, err.Error())
	}
	return owner, nil
}

func (s *nativeAIService) ListMemories(ctx context.Context, req *pb.ListMemoriesRequest) (*pb.ListMemoriesResponse, error) {
	if s.memoryService == nil || !s.memoryService.Enabled() {
		return &pb.ListMemoriesResponse{Code: 0, Msg: "success", Total: 0}, nil
	}
	owner, err := memoryOwnerFromRequest(ctx, req.GetTenantId(), req.GetOwnerRole(), req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	items, total, err := s.memoryService.List(ctx, appmemory.ListFilter{
		Owner:      owner,
		ScopeType:  req.GetScopeType(),
		ScopeID:    req.GetScopeId(),
		MemoryType: req.GetMemoryType(),
		Status:     domainmemory.Status(req.GetStatus()),
		PIILevel:   domainmemory.PIILevel(req.GetPiiLevel()),
		Page:       int(req.GetPage()),
		PageSize:   int(req.GetPageSize()),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*pb.MemoryInfo, 0, len(items))
	for _, item := range items {
		out = append(out, memoryInfoFromDomain(item))
	}
	return &pb.ListMemoriesResponse{Code: 0, Msg: "success", Total: total, List: out}, nil
}

func (s *nativeAIService) GetMemory(ctx context.Context, req *pb.GetMemoryRequest) (*pb.MemoryResponse, error) {
	if s.memoryService == nil || !s.memoryService.Enabled() {
		return &pb.MemoryResponse{Code: 404, Msg: "not found"}, nil
	}
	owner, err := memoryOwnerFromRequest(ctx, req.GetTenantId(), req.GetOwnerRole(), req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	item, found, err := s.memoryService.Get(ctx, owner, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.MemoryResponse{Code: 404, Msg: "not found"}, nil
	}
	return &pb.MemoryResponse{Code: 0, Msg: "success", Memory: memoryInfoFromDomain(item)}, nil
}

func (s *nativeAIService) CreateMemory(ctx context.Context, req *pb.CreateMemoryRequest) (*pb.MemoryResponse, error) {
	if s.memoryService == nil || !s.memoryService.Enabled() {
		return &pb.MemoryResponse{Code: 503, Msg: "memory service disabled"}, nil
	}
	owner, err := memoryOwnerFromRequest(ctx, req.GetTenantId(), req.GetOwnerRole(), req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	saved, err := s.memoryService.Write(ctx, appmemory.WriteRequest{
		Owner:          owner,
		Scope:          domainmemory.Scope{Type: req.GetScopeType(), ID: req.GetScopeId()},
		MemoryType:     req.GetMemoryType(),
		Content:        req.GetContent(),
		Source:         req.GetSource(),
		Confidence:     req.GetConfidence(),
		Importance:     req.GetImportance(),
		ConfirmHighPII: req.GetConfirmHighPii(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if saved.ID == 0 {
		return &pb.MemoryResponse{Code: 202, Msg: "accepted without persist"}, nil
	}
	return &pb.MemoryResponse{Code: 0, Msg: "success", Memory: memoryInfoFromDomain(saved)}, nil
}

func (s *nativeAIService) UpdateMemory(ctx context.Context, req *pb.UpdateMemoryRequest) (*pb.MemoryResponse, error) {
	if s.memoryService == nil || !s.memoryService.Enabled() {
		return &pb.MemoryResponse{Code: 503, Msg: "memory service disabled"}, nil
	}
	owner, err := memoryOwnerFromRequest(ctx, req.GetTenantId(), req.GetOwnerRole(), req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	existing, found, err := s.memoryService.Get(ctx, owner, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.MemoryResponse{Code: 404, Msg: "not found"}, nil
	}
	if req.GetContentSet() {
		existing.Content = req.GetContent()
	}
	if req.GetConfidenceSet() {
		existing.Confidence = req.GetConfidence()
	}
	if req.GetImportanceSet() {
		existing.Importance = req.GetImportance()
	}
	if req.GetStatusSet() {
		existing.Status = domainmemory.Status(req.GetStatus())
	}
	if req.GetPiiLevelSet() {
		existing.PIILevel = domainmemory.PIILevel(req.GetPiiLevel())
	}
	if req.GetExpiresAtSet() && strings.TrimSpace(req.GetExpiresAt()) != "" {
		parsed, parseErr := time.Parse(time.RFC3339, req.GetExpiresAt())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid expires_at")
		}
		existing.ExpiresAt = &parsed
	}
	updated, err := s.memoryService.Update(ctx, existing)
	if err != nil {
		return nil, err
	}
	return &pb.MemoryResponse{Code: 0, Msg: "success", Memory: memoryInfoFromDomain(updated)}, nil
}

func (s *nativeAIService) RevokeMemory(ctx context.Context, req *pb.RevokeMemoryRequest) (*pb.MemoryResponse, error) {
	if s.memoryService == nil || !s.memoryService.Enabled() {
		return &pb.MemoryResponse{Code: 503, Msg: "memory service disabled"}, nil
	}
	owner, err := memoryOwnerFromRequest(ctx, req.GetTenantId(), req.GetOwnerRole(), req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	if err := s.memoryService.Revoke(ctx, owner, req.GetId(), uint64(req.GetRevokedBy()), req.GetRevokeReason()); err != nil {
		return nil, err
	}
	item, found, err := s.memoryService.Get(ctx, owner, req.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.MemoryResponse{Code: 0, Msg: "success"}, nil
	}
	return &pb.MemoryResponse{Code: 0, Msg: "success", Memory: memoryInfoFromDomain(item)}, nil
}

func (s *nativeAIService) RecallMemories(ctx context.Context, req *pb.RecallMemoriesRequest) (*pb.RecallMemoriesResponse, error) {
	if s.memoryService == nil || !s.memoryService.Enabled() {
		return &pb.RecallMemoriesResponse{Code: 0, Msg: "success"}, nil
	}
	owner, err := memoryOwnerFromRequest(ctx, req.GetTenantId(), req.GetOwnerRole(), req.GetOwnerId())
	if err != nil {
		return nil, err
	}
	scopes := []domainmemory.Scope{{Type: req.GetScopeType(), ID: req.GetScopeId()}}
	if req.GetScopeType() == "" {
		scopes = nil
	}
	result, err := s.memoryService.Recall(ctx, appmemory.RecallRequest{
		Owner:           owner,
		Scopes:          scopes,
		Query:           req.GetQuery(),
		TargetScopeType: req.GetScopeType(),
		TargetScopeID:   req.GetScopeId(),
		MemoryTypes:     req.GetMemoryTypes(),
	})
	if err != nil {
		return nil, err
	}
	items := make([]*pb.RecallMemoryItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, &pb.RecallMemoryItem{
			Memory: memoryInfoFromDomain(item.Memory),
			Score:  item.Score,
			Reason: item.Reason,
		})
	}
	return &pb.RecallMemoriesResponse{Code: 0, Msg: "success", Items: items}, nil
}

func memoryInfoFromDomain(memory domainmemory.Memory) *pb.MemoryInfo {
	info := &pb.MemoryInfo{
		Id:           memory.ID,
		OwnerRole:    int32(memory.OwnerRole),
		OwnerId:      memory.OwnerID,
		HrId:         memory.HRID,
		ScopeType:    memory.Scope.Type,
		ScopeId:      memory.Scope.ID,
		MemoryType:   memory.MemoryType,
		Content:      memory.Content,
		Source:       memory.Source,
		Confidence:   memory.Confidence,
		Importance:   memory.Importance,
		Status:       string(memory.Status),
		PiiLevel:     string(memory.PIILevel),
		ContentHash:  memory.ContentHash,
		RevokeReason: memory.RevokeReason,
	}
	if memory.TenantID != nil {
		info.TenantId = int64(*memory.TenantID)
	}
	if memory.SourceSessionID != nil {
		info.SourceSessionId = int64(*memory.SourceSessionID)
	}
	if memory.SourceMessageID != nil {
		info.SourceMessageId = int64(*memory.SourceMessageID)
	}
	if memory.SourceRunID != nil {
		info.SourceRunId = int64(*memory.SourceRunID)
	}
	if memory.CreatedBy != nil {
		info.CreatedBy = int64(*memory.CreatedBy)
	}
	if memory.RevokedBy != nil {
		info.RevokedBy = int64(*memory.RevokedBy)
	}
	if memory.ExpiresAt != nil {
		info.ExpiresAt = memory.ExpiresAt.Format(time.RFC3339)
	}
	if memory.DeletedAt != nil {
		info.DeletedAt = memory.DeletedAt.Format(time.RFC3339)
	}
	if !memory.CreatedAt.IsZero() {
		info.CreatedAt = memory.CreatedAt.Format(time.RFC3339)
	}
	if !memory.UpdatedAt.IsZero() {
		info.UpdatedAt = memory.UpdatedAt.Format(time.RFC3339)
	}
	return info
}
