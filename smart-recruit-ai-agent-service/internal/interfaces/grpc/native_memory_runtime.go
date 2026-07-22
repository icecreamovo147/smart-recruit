package grpc

import (
	"context"
	"strings"
	"time"

	appmemory "smart-recruit-ai-agent-service/internal/application/memory"
	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
	"smart-recruit-proto/recruitment/pb"
)

const emptyMemorySection = "当前没有额外注入的长期记忆。"

type hrMemoryRecallSnapshot struct {
	InjectText string
	Evidence   hrRuntimeMemoryEvidence
}

func (s *nativeAIService) recallHRMemory(ctx context.Context, req *pb.ChatRequest, jobID, candidateUserID uint64) hrMemoryRecallSnapshot {
	if s == nil || s.memoryService == nil || !s.memoryService.Enabled() {
		return hrMemoryRecallSnapshot{InjectText: emptyMemorySection}
	}
	scopes := appmemory.BuildHRRecallScopes(uint64(req.GetHrId()), uint64(req.GetApplicationId()), jobID, candidateUserID)
	targetType := domainmemory.ScopeApplication
	targetID := uint64(req.GetApplicationId())
	if targetID == 0 {
		targetType = domainmemory.ScopeHR
		targetID = 0
	}
	vectorScores := s.semanticMemoryScores(ctx, domainmemory.OwnerRoleHR, uint64(req.GetHrId()), req.GetMessage(), scopes, targetType, targetID)
	result, err := s.memoryService.Recall(ctx, appmemory.RecallRequest{
		OwnerRole:       domainmemory.OwnerRoleHR,
		OwnerID:         uint64(req.GetHrId()),
		Scopes:          scopes,
		Query:           req.GetMessage(),
		TargetScopeType: targetType,
		TargetScopeID:   targetID,
		VectorScores:    vectorScores,
	})
	if err != nil || strings.TrimSpace(result.InjectText) == "" {
		return hrMemoryRecallSnapshot{InjectText: emptyMemorySection}
	}
	return hrMemoryRecallSnapshot{
		InjectText: result.InjectText,
		Evidence:   memoryEvidenceFromRecall(result.Evidence),
	}
}

func (s *nativeAIService) recallHRMemorySection(ctx context.Context, req *pb.ChatRequest, jobID, candidateUserID uint64) string {
	return s.recallHRMemory(ctx, req, jobID, candidateUserID).InjectText
}

func (s *nativeAIService) recallCandidateMemory(ctx context.Context, userID, applicationID, jobID int64, query string) appmemory.RecallResult {
	if s == nil || s.memoryService == nil || !s.memoryService.Enabled() {
		return appmemory.RecallResult{}
	}
	scopes := appmemory.BuildCandidateRecallScopes(uint64(userID), uint64(applicationID), uint64(jobID))
	targetType := domainmemory.ScopeUser
	targetID := uint64(userID)
	if applicationID > 0 {
		targetType = domainmemory.ScopeApplication
		targetID = uint64(applicationID)
	}
	vectorScores := s.semanticMemoryScores(ctx, domainmemory.OwnerRoleCandidate, uint64(userID), query, scopes, targetType, targetID)
	result, err := s.memoryService.Recall(ctx, appmemory.RecallRequest{
		OwnerRole:       domainmemory.OwnerRoleCandidate,
		OwnerID:         uint64(userID),
		Scopes:          scopes,
		Query:           query,
		TargetScopeType: targetType,
		TargetScopeID:   targetID,
		VectorScores:    vectorScores,
	})
	if err != nil {
		return appmemory.RecallResult{}
	}
	return result
}

func (s *nativeAIService) recallCandidateMemorySection(ctx context.Context, userID, applicationID, jobID int64, query string) string {
	return s.recallCandidateMemory(ctx, userID, applicationID, jobID, query).InjectText
}

func (s *nativeAIService) semanticMemoryScores(ctx context.Context, ownerRole domainmemory.OwnerRole, ownerID uint64, query string, scopes []domainmemory.Scope, targetScopeType string, targetScopeID uint64) map[uint64]float64 {
	if s == nil || s.embedding == nil || ownerID == 0 || strings.TrimSpace(query) == "" {
		return nil
	}
	scores, _ := s.embedding.SemanticMemoryScores(ctx, ownerRole, ownerID, query, scopes, targetScopeType, targetScopeID, 0)
	return scores
}

func memoryEvidenceFromRecall(evidence appmemory.RecallEvidence) hrRuntimeMemoryEvidence {
	if evidence.Count == 0 {
		return hrRuntimeMemoryEvidence{}
	}
	return hrRuntimeMemoryEvidence{
		MemoryIDs:      append([]uint64(nil), evidence.MemoryIDs...),
		Count:          evidence.Count,
		Chars:          evidence.Chars,
		RelevanceModes: append([]string(nil), evidence.RelevanceModes...),
	}
}

func (s *nativeAIService) asyncExtractHRMemory(req *pb.ChatRequest, sessionID int64, userMessage, assistantReply string, jobID, candidateUserID uint64) {
	if s == nil || s.memoryService == nil || !s.memoryService.WriteEnabled() {
		return
	}
	go func() {
		defer func() { _ = recover() }()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		scopes := appmemory.BuildHRRecallScopes(uint64(req.GetHrId()), uint64(req.GetApplicationId()), jobID, candidateUserID)
		sessionIDValue := uint64(sessionID)
		createdBy := uint64(req.GetHrId())
		_ = s.memoryService.WriteFromExtractor(ctx, appmemory.ExtractInput{
			OwnerRole: domainmemory.OwnerRoleHR,
			OwnerID:   uint64(req.GetHrId()),
			UserText:  userMessage,
			ReplyText: assistantReply,
		}, scopes, appmemory.WriteRequest{
			SourceSessionID: &sessionIDValue,
			CreatedBy:       &createdBy,
		})
	}()
}

func (s *nativeAIService) asyncExtractCandidateMemory(userID, sessionID, applicationID, jobID int64, userMessage, assistantReply string) {
	if s == nil || s.memoryService == nil || !s.memoryService.WriteEnabled() {
		return
	}
	go func() {
		defer func() { _ = recover() }()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		scopes := appmemory.BuildCandidateRecallScopes(uint64(userID), uint64(applicationID), uint64(jobID))
		sessionIDValue := uint64(sessionID)
		createdBy := uint64(userID)
		_ = s.memoryService.WriteFromExtractor(ctx, appmemory.ExtractInput{
			OwnerRole: domainmemory.OwnerRoleCandidate,
			OwnerID:   uint64(userID),
			UserText:  userMessage,
			ReplyText: assistantReply,
		}, scopes, appmemory.WriteRequest{
			SourceSessionID: &sessionIDValue,
			CreatedBy:       &createdBy,
		})
	}()
}

func hrRuntimePromptVariablesWithMemory(req *pb.ChatRequest, memorySection string) map[string]string {
	vars := hrRuntimePromptVariables(req)
	if strings.TrimSpace(memorySection) == "" {
		memorySection = emptyMemorySection
	}
	vars["memory_section"] = memorySection
	return vars
}

func appendMemorySectionToSystemPrompt(base, memorySection string) string {
	base = strings.TrimSpace(base)
	memorySection = strings.TrimSpace(memorySection)
	if memorySection == "" || memorySection == emptyMemorySection {
		return base
	}
	if base == "" {
		return "## 长期记忆\n" + memorySection
	}
	return base + "\n\n## 长期记忆\n" + memorySection
}

func estimateMemoryTokens(memorySection string) int32 {
	if strings.TrimSpace(memorySection) == "" || memorySection == emptyMemorySection {
		return 0
	}
	return saturatingInt32(int64(estimateTokensConservative(memorySection)))
}

func candidateJobIDFromSession(session ChatSessionRow) int64 {
	if strings.EqualFold(strings.TrimSpace(session.SourceType), "job") && session.SourceID > 0 {
		return session.SourceID
	}
	return 0
}
