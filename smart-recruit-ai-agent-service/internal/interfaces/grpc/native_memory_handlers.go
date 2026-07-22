package grpc

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	appmemory "smart-recruit-ai-agent-service/internal/application/memory"
	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
	"smart-recruit-proto/recruitment/pb"
)

func (s *nativeAIService) ListMemories(ctx context.Context, req *pb.ListMemoriesRequest) (*pb.ListMemoriesResponse, error) {
	if s.memoryService == nil || !s.memoryService.Enabled() {
		return &pb.ListMemoriesResponse{Code: 0, Msg: "success", Total: 0}, nil
	}
	if req.GetOwnerId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "owner_id is required")
	}
	items, total, err := s.memoryService.List(ctx, appmemory.ListFilter{
		OwnerRole:  domainmemory.OwnerRole(req.GetOwnerRole()),
		OwnerID:    req.GetOwnerId(),
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
	item, found, err := s.memoryService.Get(ctx, domainmemory.OwnerRole(req.GetOwnerRole()), req.GetOwnerId(), req.GetId())
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
	saved, err := s.memoryService.Write(ctx, appmemory.WriteRequest{
		OwnerRole:      domainmemory.OwnerRole(req.GetOwnerRole()),
		OwnerID:        req.GetOwnerId(),
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
	existing, found, err := s.memoryService.Get(ctx, domainmemory.OwnerRole(req.GetOwnerRole()), req.GetOwnerId(), req.GetId())
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
	if err := s.memoryService.Revoke(ctx, domainmemory.OwnerRole(req.GetOwnerRole()), req.GetOwnerId(), req.GetId(), uint64(req.GetRevokedBy()), req.GetRevokeReason()); err != nil {
		return nil, err
	}
	item, found, err := s.memoryService.Get(ctx, domainmemory.OwnerRole(req.GetOwnerRole()), req.GetOwnerId(), req.GetId())
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
	scopes := []domainmemory.Scope{{Type: req.GetScopeType(), ID: req.GetScopeId()}}
	if req.GetScopeType() == "" {
		scopes = nil
	}
	result, err := s.memoryService.Recall(ctx, appmemory.RecallRequest{
		OwnerRole:       domainmemory.OwnerRole(req.GetOwnerRole()),
		OwnerID:         req.GetOwnerId(),
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
		Id:         memory.ID,
		OwnerRole:  int32(memory.OwnerRole),
		OwnerId:    memory.OwnerID,
		HrId:       memory.HRID,
		ScopeType:  memory.Scope.Type,
		ScopeId:    memory.Scope.ID,
		MemoryType: memory.MemoryType,
		Content:    memory.Content,
		Source:     memory.Source,
		Confidence: memory.Confidence,
		Importance: memory.Importance,
		Status:     string(memory.Status),
		PiiLevel:   string(memory.PIILevel),
		ContentHash: memory.ContentHash,
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
